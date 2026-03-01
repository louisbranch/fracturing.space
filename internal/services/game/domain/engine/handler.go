package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

var (
	// ErrCommandRegistryRequired indicates a missing command registry.
	ErrCommandRegistryRequired = errors.New("command registry is required")
	// ErrCommandMustMutate indicates a command returned no mutations.
	ErrCommandMustMutate = errors.New("command must emit at least one event")
	// ErrDeciderRequired indicates a missing decider.
	ErrDeciderRequired = errors.New("decider is required")
	// ErrEventRegistryRequired indicates a missing event registry.
	ErrEventRegistryRequired = errors.New("event registry is required")
	// ErrJournalRequired indicates a missing event journal.
	ErrJournalRequired = errors.New("event journal is required")
	// ErrGateStateLoaderRequired indicates a missing gate state loader.
	ErrGateStateLoaderRequired = errors.New("gate state loader is required")
	// ErrPostPersistApplyFailed indicates that events were persisted to the
	// journal but the in-memory fold step failed. Callers should use
	// errors.Is to detect this condition and recover via replay rather than
	// retrying the command, which would create duplicates.
	ErrPostPersistApplyFailed = errors.New("post-persist fold failed")
)

// GateStateLoader loads session state for gate checks.
//
// It lets the command flow remain read-light by loading only session context when
// gate policy evaluation is required.
type GateStateLoader interface {
	LoadSession(ctx context.Context, campaignID, sessionID string) (session.State, error)
}

// StateLoader loads domain state for deciders.
//
// This keeps deciders pure: they never fetch data themselves and always operate
// on fully reconstructed state from the replay pipeline.
type StateLoader interface {
	Load(ctx context.Context, cmd command.Command) (any, error)
}

// EventJournal appends events to the journal.
//
// Appending here is the persistence boundary for the write model.
// BatchAppend guarantees that all events from a single command decision are
// persisted atomically in one transaction.
type EventJournal interface {
	Append(ctx context.Context, evt event.Event) (event.Event, error)
	BatchAppend(ctx context.Context, events []event.Event) ([]event.Event, error)
}

// Folder folds events into aggregate state.
//
// The folder is intentionally shared between request-time execution and replay
// so behavior is identical when handling new commands and reconstructing history.
// Named "Folder" (not "Applier") to distinguish pure state folds from
// projection.Applier, which performs side-effecting I/O writes to stores.
type Folder interface {
	Fold(state any, evt event.Event) (any, error)
}

// Decider returns a decision for a command.
//
// Implementations are where business invariants live; handlers just orchestrate
// transport-safe execution.
type Decider interface {
	Decide(state any, cmd command.Command, now func() time.Time) command.Decision
}

// Handler is the domain write orchestrator:
// 1) validate intent against command registry,
// 2) enforce optional session gate policy,
// 3) execute deciders over replay-derived state,
// 4) validate events against event registry,
// 5) append events to the journal,
// 6) apply events to in-memory state,
// 7) checkpoint and snapshot state for fast future replays.
type Handler struct {
	Commands        *command.Registry
	Events          *event.Registry
	Journal         EventJournal
	Checkpoints     replay.CheckpointStore
	Snapshots       StateSnapshotStore
	Gate            DecisionGate
	GateStateLoader GateStateLoader
	StateLoader     StateLoader
	Decider         Decider
	Folder          Folder
	Now             func() time.Time
}

// HandlerConfig holds the dependencies for constructing a Handler.
type HandlerConfig struct {
	Commands        *command.Registry
	Events          *event.Registry
	Journal         EventJournal
	Checkpoints     replay.CheckpointStore
	Snapshots       StateSnapshotStore
	Gate            DecisionGate
	GateStateLoader GateStateLoader
	StateLoader     StateLoader
	Decider         Decider
	Folder          Folder
	Now             func() time.Time
}

// NewHandler validates required dependencies and returns a configured Handler.
// Use this constructor in production wiring to catch missing dependencies at
// startup rather than at first request. The Handler struct remains exported
// for test flexibility where only a subset of fields is needed.
func NewHandler(cfg HandlerConfig) (Handler, error) {
	if cfg.Commands == nil {
		return Handler{}, ErrCommandRegistryRequired
	}
	if cfg.Events == nil {
		return Handler{}, ErrEventRegistryRequired
	}
	if cfg.Journal == nil {
		return Handler{}, ErrJournalRequired
	}
	if cfg.Decider == nil {
		return Handler{}, ErrDeciderRequired
	}
	return Handler{
		Commands:        cfg.Commands,
		Events:          cfg.Events,
		Journal:         cfg.Journal,
		Checkpoints:     cfg.Checkpoints,
		Snapshots:       cfg.Snapshots,
		Gate:            cfg.Gate,
		GateStateLoader: cfg.GateStateLoader,
		StateLoader:     cfg.StateLoader,
		Decider:         cfg.Decider,
		Folder:          cfg.Folder,
		Now:             cfg.Now,
	}, nil
}

// Result captures execution outcomes.
//
// Result captures both the command decision and any newly folded in-memory state so
// transport layers can support read-after-write flows without a second load.
type Result struct {
	Decision command.Decision
	State    any
}

// Handle validates a command, checks gate policy, and returns a decision.
//
// Use Handle when you need validation plus event emission decisions without
// requiring caller to materialize post-apply state.
func (h Handler) Handle(ctx context.Context, cmd command.Command) (command.Decision, error) {
	_, _, decision, err := h.prepareExecution(ctx, cmd)
	if err != nil {
		return command.Decision{}, err
	}
	return decision, nil
}

func (h Handler) Execute(ctx context.Context, cmd command.Command) (Result, error) {
	validated, state, decision, err := h.prepareExecution(ctx, cmd)
	if err != nil {
		return Result{}, err
	}
	now := h.nowFunc()
	if h.Checkpoints != nil && len(decision.Events) > 0 {
		last := decision.Events[len(decision.Events)-1]
		if last.Seq > 0 {
			if err := h.Checkpoints.Save(ctx, replay.Checkpoint{
				CampaignID: validated.CampaignID,
				LastSeq:    last.Seq,
				UpdatedAt:  now().UTC(),
			}); err != nil {
				return Result{}, err
			}
		}
	}
	if h.Snapshots != nil && len(decision.Events) > 0 {
		last := decision.Events[len(decision.Events)-1]
		if last.Seq > 0 {
			if err := h.Snapshots.SaveState(ctx, validated.CampaignID, last.Seq, state); err != nil {
				return Result{}, err
			}
		}
	}
	return Result{Decision: decision, State: state}, nil
}

func (h Handler) prepareExecution(ctx context.Context, cmd command.Command) (command.Command, any, command.Decision, error) {
	validated, err := h.validateCommand(cmd)
	if err != nil {
		return command.Command{}, nil, command.Decision{}, err
	}

	gateDecision, shortCircuit, err := h.evaluateSessionGate(ctx, validated)
	if err != nil {
		return command.Command{}, nil, command.Decision{}, err
	}
	if shortCircuit {
		return validated, nil, gateDecision, nil
	}

	state, err := h.loadState(ctx, validated)
	if err != nil {
		return command.Command{}, nil, command.Decision{}, err
	}

	decision, err := h.decide(state, validated)
	if err != nil {
		return command.Command{}, nil, command.Decision{}, err
	}

	decision, err = h.validateDecisionEvents(decision)
	if err != nil {
		return command.Command{}, nil, command.Decision{}, err
	}

	decision, err = h.appendDecisionEvents(ctx, decision)
	if err != nil {
		return command.Command{}, nil, command.Decision{}, err
	}

	state, err = h.applyDecisionEvents(state, decision)
	if err != nil {
		return command.Command{}, nil, command.Decision{}, err
	}

	return validated, state, decision, nil
}

func (h Handler) validateCommand(cmd command.Command) (command.Command, error) {
	// Required in production via NewHandler; nil here supports test-path flexibility.
	if h.Commands == nil {
		return command.Command{}, ErrCommandRegistryRequired
	}
	return h.Commands.ValidateForDecision(cmd)
}

func (h Handler) evaluateSessionGate(ctx context.Context, cmd command.Command) (command.Decision, bool, error) {
	def, ok := h.Commands.Definition(cmd.Type)
	if !ok || def.Gate.Scope != command.GateScopeSession {
		return command.Decision{}, false, nil
	}

	// Truly optional; only required for gate-scoped commands.
	if h.GateStateLoader == nil {
		return command.Decision{}, false, ErrGateStateLoaderRequired
	}
	state, err := h.GateStateLoader.LoadSession(ctx, cmd.CampaignID, cmd.SessionID)
	if err != nil {
		return command.Decision{}, false, err
	}
	decision := h.Gate.Check(state, cmd)
	if len(decision.Rejections) > 0 {
		return decision, true, nil
	}
	return command.Decision{}, false, nil
}

func (h Handler) loadState(ctx context.Context, cmd command.Command) (any, error) {
	var state any
	// Truly optional; falls back to nil state (stateless deciders).
	if h.StateLoader == nil {
		return state, nil
	}
	return h.StateLoader.Load(ctx, cmd)
}

func (h Handler) decide(state any, cmd command.Command) (command.Decision, error) {
	// Required in production via NewHandler; nil here supports test-path flexibility.
	if h.Decider == nil {
		return command.Decision{}, ErrDeciderRequired
	}

	now := h.nowFunc()
	decision := h.Decider.Decide(state, cmd, now)
	if err := decision.Validate(); err != nil {
		// FIXME(telemetry): emit metric for command decider no-op outcomes (no events, no rejections)
		// once domain/write model counters are wired.
		return command.Decision{}, ErrCommandMustMutate
	}
	return decision, nil
}

func (h Handler) nowFunc() func() time.Time {
	now := h.Now
	if now == nil {
		now = time.Now
	}
	return now
}

func (h Handler) validateDecisionEvents(decision command.Decision) (command.Decision, error) {
	// Required in production via NewHandler; nil here supports test-path flexibility.
	if h.Events == nil || len(decision.Events) == 0 {
		return decision, nil
	}
	validated := make([]event.Event, 0, len(decision.Events))
	for _, evt := range decision.Events {
		vetted, err := h.Events.ValidateForAppend(evt)
		if err != nil {
			return command.Decision{}, err
		}
		validated = append(validated, vetted)
	}
	decision.Events = validated
	return decision, nil
}

func (h Handler) appendDecisionEvents(ctx context.Context, decision command.Decision) (command.Decision, error) {
	// Required in production via NewHandler; nil here supports test-path flexibility.
	if h.Journal == nil || len(decision.Events) == 0 {
		return decision, nil
	}
	stored, err := h.Journal.BatchAppend(ctx, decision.Events)
	if err != nil {
		return command.Decision{}, err
	}
	decision.Events = stored
	return decision, nil
}

func (h Handler) applyDecisionEvents(state any, decision command.Decision) (any, error) {
	// Required in production via NewHandler; nil here supports test-path flexibility.
	if h.Folder == nil || len(decision.Events) == 0 {
		return state, nil
	}
	journalPersisted := h.Journal != nil
	for _, evt := range decision.Events {
		stateAfter, err := h.Folder.Fold(state, evt)
		if err != nil {
			if journalPersisted {
				return nil, wrapNonRetryable(fmt.Errorf("%w: %w", ErrPostPersistApplyFailed, err))
			}
			return nil, err
		}
		state = stateAfter
	}
	return state, nil
}
