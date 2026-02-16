package engine

import (
	"context"
	"errors"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

var (
	// ErrCommandRegistryRequired indicates a missing command registry.
	ErrCommandRegistryRequired = errors.New("command registry is required")
	// ErrDeciderRequired indicates a missing decider.
	ErrDeciderRequired = errors.New("decider is required")
	// ErrGateStateLoaderRequired indicates a missing gate state loader.
	ErrGateStateLoaderRequired = errors.New("gate state loader is required")
)

// GateStateLoader loads session state for gate checks.
type GateStateLoader interface {
	LoadSession(ctx context.Context, campaignID, sessionID string) (session.State, error)
}

// StateLoader loads domain state for deciders.
type StateLoader interface {
	Load(ctx context.Context, cmd command.Command) (any, error)
}

// EventJournal appends events to the journal.
type EventJournal interface {
	Append(ctx context.Context, evt event.Event) (event.Event, error)
}

// Applier folds events into state.
type Applier interface {
	Apply(state any, evt event.Event) (any, error)
}

// Decider returns a decision for a command.
type Decider interface {
	Decide(state any, cmd command.Command, now func() time.Time) command.Decision
}

// Handler validates, gates, and decides commands.
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
	Applier         Applier
	Now             func() time.Time
}

// Result captures execution outcomes.
type Result struct {
	Decision command.Decision
	State    any
}

// Handle validates a command, checks gate policy, and returns a decision.
func (h Handler) Handle(ctx context.Context, cmd command.Command) (command.Decision, error) {
	if h.Commands == nil {
		return command.Decision{}, ErrCommandRegistryRequired
	}
	validated, err := h.Commands.ValidateForDecision(cmd)
	if err != nil {
		return command.Decision{}, err
	}
	cmd = validated

	if def, ok := h.Commands.Definition(cmd.Type); ok && def.Gate.Scope == command.GateScopeSession {
		if h.GateStateLoader == nil {
			return command.Decision{}, ErrGateStateLoaderRequired
		}
		state, err := h.GateStateLoader.LoadSession(ctx, cmd.CampaignID, cmd.SessionID)
		if err != nil {
			return command.Decision{}, err
		}
		decision := h.Gate.Check(state, cmd)
		if len(decision.Rejections) > 0 {
			return decision, nil
		}
	}

	if h.Decider == nil {
		return command.Decision{}, ErrDeciderRequired
	}
	var state any
	if h.StateLoader != nil {
		state, err = h.StateLoader.Load(ctx, cmd)
		if err != nil {
			return command.Decision{}, err
		}
	}
	now := h.Now
	if now == nil {
		now = time.Now
	}
	decision := h.Decider.Decide(state, cmd, now)
	if h.Events != nil && len(decision.Events) > 0 {
		validated := make([]event.Event, 0, len(decision.Events))
		for _, evt := range decision.Events {
			vetted, err := h.Events.ValidateForAppend(evt)
			if err != nil {
				return command.Decision{}, err
			}
			validated = append(validated, vetted)
		}
		decision.Events = validated
	}
	if h.Journal != nil && len(decision.Events) > 0 {
		stored := make([]event.Event, 0, len(decision.Events))
		for _, evt := range decision.Events {
			appended, err := h.Journal.Append(ctx, evt)
			if err != nil {
				return command.Decision{}, err
			}
			stored = append(stored, appended)
		}
		decision.Events = stored
	}
	return decision, nil
}

// Execute handles a command and applies emitted events to state.
func (h Handler) Execute(ctx context.Context, cmd command.Command) (Result, error) {
	normalized := cmd
	if h.Commands != nil {
		validated, err := h.Commands.ValidateForDecision(cmd)
		if err != nil {
			return Result{}, err
		}
		normalized = validated
	}

	decision, err := h.Handle(ctx, normalized)
	if err != nil {
		return Result{}, err
	}
	var state any
	if h.StateLoader != nil {
		state, err = h.StateLoader.Load(ctx, normalized)
		if err != nil {
			return Result{}, err
		}
	}
	loadedState := state
	if h.Applier != nil && len(decision.Events) > 0 {
		for _, evt := range decision.Events {
			state, err = h.Applier.Apply(state, evt)
			if err != nil {
				return Result{}, err
			}
		}
	}
	if h.Checkpoints != nil && len(decision.Events) > 0 {
		last := decision.Events[len(decision.Events)-1]
		if last.Seq > 0 {
			if err := h.Checkpoints.Save(ctx, replay.Checkpoint{
				CampaignID: normalized.CampaignID,
				LastSeq:    last.Seq,
				UpdatedAt:  time.Now().UTC(),
			}); err != nil {
				return Result{}, err
			}
		}
	}
	if h.Snapshots != nil && len(decision.Events) > 0 {
		last := decision.Events[len(decision.Events)-1]
		if last.Seq > 0 {
			snapshotState := state
			if h.Journal != nil && h.StateLoader != nil {
				// When events were appended before state load, loadedState already
				// includes those events exactly once.
				snapshotState = loadedState
			}
			if err := h.Snapshots.SaveState(ctx, normalized.CampaignID, last.Seq, snapshotState); err != nil {
				return Result{}, err
			}
		}
	}
	return Result{Decision: decision, State: state}, nil
}
