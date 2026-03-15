package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

const rejectionCodeSessionAITurnNotActive = "SESSION_AI_TURN_NOT_ACTIVE"

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
	// ErrSceneGateStateLoaderRequired indicates a missing scene gate state loader.
	ErrSceneGateStateLoaderRequired = errors.New("scene gate state loader is required")
	// ErrSceneIDRequired indicates a missing scene id for scene-scoped commands.
	ErrSceneIDRequired = errors.New("scene id is required for scene-scoped command")
	// ErrPostPersistApplyFailed indicates that events were persisted to the
	// journal but the in-memory fold step failed. Callers should use
	// errors.Is to detect this condition and recover via replay rather than
	// retrying the command, which would create duplicates.
	ErrPostPersistApplyFailed = errors.New("post-persist fold failed")
	// ErrPostPersistSnapshotFailed indicates snapshot persistence failed after
	// journal append succeeded.
	ErrPostPersistSnapshotFailed = errors.New("post-persist snapshot save failed")
	// ErrPostPersistCheckpointFailed indicates checkpoint persistence failed
	// after journal append succeeded.
	ErrPostPersistCheckpointFailed = errors.New("post-persist checkpoint save failed")
	// ErrStateFactoryRequired indicates a missing state factory.
	ErrStateFactoryRequired = errors.New("state factory is required")
)

// GateStateLoader loads session state for gate checks.
//
// It lets the command flow remain read-light by loading only session context when
// gate policy evaluation is required.
type GateStateLoader interface {
	LoadSession(ctx context.Context, campaignID, sessionID string) (session.State, error)
}

// SceneGateStateLoader loads scene state for scene-scoped gate checks.
type SceneGateStateLoader interface {
	LoadScene(ctx context.Context, campaignID, sceneID string) (scene.State, error)
}

// StateLoader loads domain state for deciders.
//
// This keeps deciders pure: they never fetch data themselves and always operate
// on fully reconstructed state from the replay pipeline.
type StateLoader interface {
	Load(ctx context.Context, cmd command.Command) (any, error)
}

// FreshStateLoader optionally reconstructs state without cached snapshots or
// checkpoints. Handlers use this to verify rejections against authoritative
// journal replay before returning them to callers.
type FreshStateLoader interface {
	LoadFresh(ctx context.Context, cmd command.Command) (any, error)
}

// EventJournal appends events to the journal.
//
// Appending here is the persistence boundary for the write model.
// BatchAppend guarantees that all events from a single command decision are
// persisted atomically in one transaction.
type EventJournal interface {
	BatchAppend(ctx context.Context, events []event.Event) ([]event.Event, error)
}

// Folder folds events into aggregate state.
//
// The folder is intentionally shared between request-time execution and replay
// so behavior is identical when handling new commands and reconstructing history.
// Named "Folder" (not "Applier") to distinguish pure state folds from
// projection.Applier, which performs side-effecting I/O writes to stores.
//
// Intentionally defined at the consumption point (Go interface-at-consumer
// pattern). Parallel definitions exist at:
//   - domain/replay.Folder (replay path)
//   - domain/module.Folder (adds FoldHandledTypes for system fold coverage)
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
//  1. validate intent against command registry,
//  2. enforce optional session gate policy,
//  3. execute deciders over replay-derived state,
//  4. validate events against event registry,
//  5. append events to the journal,
//  6. apply events to in-memory state,
//  7. checkpoint and snapshot state for fast future replays.
//
// Required fields (validated by [NewHandler]):
//   - Commands, Events, Journal, Decider
//   - GateStateLoader — required when any registered command uses [command.GateScopeSession]
//   - SceneGateStateLoader — required when any registered command uses [command.GateScopeScene]
//
// Optional fields (nil-safe at call sites):
//   - Checkpoints, Snapshots — enable fast replay resume; omit to replay from the beginning
//   - StateLoader — loads reconstructed state; nil skips state loading (useful in stateless test harnesses)
//   - Folder — folds events into state after append; nil skips in-memory fold
//   - Now — clock source; defaults to [time.Now] when nil
//   - Gate — auto-configured by [NewHandler] from Commands; callers should not set Gate.Registry directly
type Handler struct {
	Commands             *command.Registry
	Events               *event.Registry
	Journal              EventJournal
	Checkpoints          replay.CheckpointStore
	Snapshots            StateSnapshotStore
	Gate                 DecisionGate
	GateStateLoader      GateStateLoader
	SceneGateStateLoader SceneGateStateLoader
	StateLoader          StateLoader
	Decider              Decider
	Folder               Folder
	Now                  func() time.Time
}

// NewHandler validates required dependencies and returns a configured Handler.
// Use this constructor in production wiring to catch missing dependencies at
// startup rather than at first request. The Handler struct remains exported
// for test flexibility where only a subset of fields is needed.
func NewHandler(h Handler) (Handler, error) {
	if h.Commands == nil {
		return Handler{}, ErrCommandRegistryRequired
	}
	if h.Events == nil {
		return Handler{}, ErrEventRegistryRequired
	}
	if h.Journal == nil {
		return Handler{}, ErrJournalRequired
	}
	if h.Decider == nil {
		return Handler{}, ErrDeciderRequired
	}
	if requiresGateScope(h.Commands, command.GateScopeSession) && h.GateStateLoader == nil {
		return Handler{}, ErrGateStateLoaderRequired
	}
	if requiresGateScope(h.Commands, command.GateScopeScene) && h.SceneGateStateLoader == nil {
		return Handler{}, ErrSceneGateStateLoaderRequired
	}
	// Bind gate registry from Commands to avoid drift and fail-open checks.
	h.Gate.Registry = h.Commands
	return h, nil
}

// Result captures execution outcomes.
//
// Result captures both the command decision and any newly folded in-memory state so
// transport layers can support read-after-write flows without a second load.
type Result struct {
	Decision command.Decision
	State    any
}

func (h Handler) Execute(ctx context.Context, cmd command.Command) (Result, error) {
	validated, state, decision, err := h.prepareExecution(ctx, cmd)
	if err != nil {
		return Result{}, err
	}
	lastSeq := lastDecisionSeq(decision.Events)
	if lastSeq == 0 {
		return Result{Decision: decision, State: state}, nil
	}

	// Capture a single post-persist timestamp for both snapshot and checkpoint
	// so they share one consistent clock reading.
	postPersistTime := h.nowFunc()().UTC()

	if h.Snapshots != nil {
		if err := h.Snapshots.SaveState(ctx, string(validated.CampaignID), lastSeq, state); err != nil {
			return Result{}, newPostPersistError(
				PostPersistStageSnapshot,
				string(validated.CampaignID),
				lastSeq,
				fmt.Errorf("%w: %w", ErrPostPersistSnapshotFailed, err),
			)
		}
	}
	if h.Checkpoints != nil {
		if err := h.Checkpoints.Save(ctx, replay.Checkpoint{
			CampaignID: string(validated.CampaignID),
			LastSeq:    lastSeq,
			UpdatedAt:  postPersistTime,
		}); err != nil {
			return Result{}, newPostPersistError(
				PostPersistStageCheckpoint,
				string(validated.CampaignID),
				lastSeq,
				fmt.Errorf("%w: %w", ErrPostPersistCheckpointFailed, err),
			)
		}
	}
	return Result{Decision: decision, State: state}, nil
}

// prepareExecution runs the full command pipeline: validate → gate → load → decide → append → fold.
//
// Gate evaluation order is intentional:
//  1. Session gate first — cheapest check, broadest scope. If the session is
//     gated (e.g. waiting for a roll outcome), most commands are rejected without
//     loading state or evaluating scene context.
//  2. Scene gate second — narrower scope, still cheaper than full state load.
//     Only evaluated for commands with GateScopeScene.
//  3. State load and decision — only reached when gates pass.
//
// This ordering minimizes work for the common rejection case (command sent while
// a gate is active) and ensures gate checks use lightweight state loaders rather
// than the full replay pipeline.
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

	sceneGateDecision, shortCircuit, err := h.evaluateSceneGate(ctx, validated)
	if err != nil {
		return command.Command{}, nil, command.Decision{}, err
	}
	if shortCircuit {
		return validated, nil, sceneGateDecision, nil
	}

	state, err := h.loadState(ctx, validated)
	if err != nil {
		return command.Command{}, nil, command.Decision{}, err
	}

	decision, err := h.decide(state, validated)
	if err != nil {
		return command.Command{}, nil, command.Decision{}, err
	}
	state, decision, err = h.retryRejectedDecisionWithFreshState(ctx, validated, state, decision)
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
	return h.evaluateGate(ctx, cmd, command.GateScopeSession, func() (command.Decision, error) {
		if h.GateStateLoader == nil {
			return command.Decision{}, ErrGateStateLoaderRequired
		}
		state, err := h.GateStateLoader.LoadSession(ctx, string(cmd.CampaignID), cmd.SessionID.String())
		if err != nil {
			return command.Decision{}, err
		}
		return h.boundGate().Check(state, cmd), nil
	})
}

func (h Handler) evaluateSceneGate(ctx context.Context, cmd command.Command) (command.Decision, bool, error) {
	return h.evaluateGate(ctx, cmd, command.GateScopeScene, func() (command.Decision, error) {
		if cmd.SceneID == "" {
			return command.Decision{}, ErrSceneIDRequired
		}
		if h.SceneGateStateLoader == nil {
			return command.Decision{}, ErrSceneGateStateLoaderRequired
		}
		state, err := h.SceneGateStateLoader.LoadScene(ctx, string(cmd.CampaignID), cmd.SceneID.String())
		if err != nil {
			return command.Decision{}, err
		}
		return h.boundGate().CheckScene(state, cmd), nil
	})
}

// evaluateGate is a shared gate evaluation flow: check scope applicability,
// delegate to loader+checker, and short-circuit on rejections.
func (h Handler) evaluateGate(_ context.Context, cmd command.Command, scope command.GateScope, check func() (command.Decision, error)) (command.Decision, bool, error) {
	def, ok := h.Commands.Definition(cmd.Type)
	if !ok || def.Gate.Scope != scope {
		return command.Decision{}, false, nil
	}
	decision, err := check()
	if err != nil {
		return command.Decision{}, false, err
	}
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
		return command.Decision{}, fmt.Errorf("%w: %w", ErrCommandMustMutate, err)
	}
	return decision, nil
}

func (h Handler) retryRejectedDecisionWithFreshState(
	ctx context.Context,
	cmd command.Command,
	state any,
	decision command.Decision,
) (any, command.Decision, error) {
	if !shouldRetryRejectedDecisionWithFreshState(cmd, decision) {
		return state, decision, nil
	}
	freshLoader, ok := h.StateLoader.(FreshStateLoader)
	if !ok {
		return state, decision, nil
	}
	freshState, err := freshLoader.LoadFresh(ctx, cmd)
	if err != nil {
		return nil, command.Decision{}, err
	}
	freshDecision, err := h.decide(freshState, cmd)
	if err != nil {
		return nil, command.Decision{}, err
	}
	return freshState, freshDecision, nil
}

func shouldRetryRejectedDecisionWithFreshState(cmd command.Command, decision command.Decision) bool {
	if len(decision.Rejections) == 0 {
		return false
	}
	if !strings.HasPrefix(strings.TrimSpace(string(cmd.Type)), "session.ai_turn.") {
		return false
	}
	for _, rejection := range decision.Rejections {
		if strings.TrimSpace(rejection.Code) != rejectionCodeSessionAITurnNotActive {
			return false
		}
	}
	return true
}

func (h Handler) nowFunc() func() time.Time {
	now := h.Now
	if now == nil {
		now = time.Now
	}
	return now
}

func (h Handler) boundGate() DecisionGate {
	gate := h.Gate
	if gate.Registry == nil {
		gate.Registry = h.Commands
	}
	return gate
}

func requiresGateScope(registry *command.Registry, scope command.GateScope) bool {
	for _, def := range registry.ListDefinitions() {
		if def.Gate.Scope == scope {
			return true
		}
	}
	return false
}

func lastDecisionSeq(events []event.Event) uint64 {
	if len(events) == 0 {
		return 0
	}
	return events[len(events)-1].Seq
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
				return nil, newPostPersistError(
					PostPersistStageFold,
					string(evt.CampaignID),
					evt.Seq,
					fmt.Errorf("%w: %w", ErrPostPersistApplyFailed, err),
				)
			}
			return nil, err
		}
		state = stateAfter
	}
	return state, nil
}
