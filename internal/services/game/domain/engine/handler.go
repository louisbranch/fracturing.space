package engine

import (
	"context"
	"errors"
	"reflect"
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
	// ErrGateStateLoaderRequired indicates a missing gate state loader.
	ErrGateStateLoaderRequired = errors.New("gate state loader is required")
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
type EventJournal interface {
	Append(ctx context.Context, evt event.Event) (event.Event, error)
}

// Applier folds events into state.
//
// The applier is intentionally shared between request-time execution and replay
// so behavior is identical when handling new commands and reconstructing history.
type Applier interface {
	Apply(state any, evt event.Event) (any, error)
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
// 4) validate and append domain events,
// 5) checkpoint and snapshot state for fast future replays.
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
	if h.Checkpoints != nil && len(decision.Events) > 0 {
		last := decision.Events[len(decision.Events)-1]
		if last.Seq > 0 {
			if err := h.Checkpoints.Save(ctx, replay.Checkpoint{
				CampaignID: validated.CampaignID,
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
			if err := h.Snapshots.SaveState(ctx, validated.CampaignID, last.Seq, state); err != nil {
				return Result{}, err
			}
		}
	}
	return Result{Decision: decision, State: state}, nil
}

func (h Handler) prepareExecution(ctx context.Context, cmd command.Command) (command.Command, any, command.Decision, error) {
	if h.Commands == nil {
		return command.Command{}, nil, command.Decision{}, ErrCommandRegistryRequired
	}
	validated, err := h.Commands.ValidateForDecision(cmd)
	if err != nil {
		return command.Command{}, nil, command.Decision{}, err
	}
	cmd = validated

	if def, ok := h.Commands.Definition(cmd.Type); ok && def.Gate.Scope == command.GateScopeSession {
		if h.GateStateLoader == nil {
			return command.Command{}, nil, command.Decision{}, ErrGateStateLoaderRequired
		}
		state, err := h.GateStateLoader.LoadSession(ctx, cmd.CampaignID, cmd.SessionID)
		if err != nil {
			return command.Command{}, nil, command.Decision{}, err
		}
		decision := h.Gate.Check(state, cmd)
		if len(decision.Rejections) > 0 {
			return cmd, nil, decision, nil
		}
	}

	if h.Decider == nil {
		return command.Command{}, nil, command.Decision{}, ErrDeciderRequired
	}
	var state any
	if h.StateLoader != nil {
		state, err = h.StateLoader.Load(ctx, cmd)
		if err != nil {
			return command.Command{}, nil, command.Decision{}, err
		}
	}
	now := h.Now
	if now == nil {
		now = time.Now
	}
	decision := h.Decider.Decide(state, cmd, now)
	if len(decision.Rejections) == 0 && len(decision.Events) == 0 {
		// FIXME(telemetry): emit metric for command decider no-op outcomes (no events, no rejections)
		// once domain/write model counters are wired.
		return command.Command{}, nil, command.Decision{}, ErrCommandMustMutate
	}
	if h.Applier != nil && len(decision.Events) > 0 {
		for _, evt := range decision.Events {
			stateAfter, err := h.Applier.Apply(state, evt)
			if err != nil {
				return command.Command{}, nil, command.Decision{}, err
			}
			state = stateAfter
		}
	}
	if h.Events != nil && len(decision.Events) > 0 {
		validated := make([]event.Event, 0, len(decision.Events))
		for _, evt := range decision.Events {
			vetted, err := h.Events.ValidateForAppend(evt)
			if err != nil {
				return command.Command{}, nil, command.Decision{}, err
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
				return command.Command{}, nil, command.Decision{}, err
			}
			stored = append(stored, appended)
		}
		decision.Events = stored
	}
	return cmd, state, decision, nil
}

func cloneState(state any) (any, error) {
	if state == nil {
		return nil, nil
	}
	cloned, err := cloneValue(reflect.ValueOf(state))
	if err != nil {
		return nil, err
	}
	return cloned.Interface(), nil
}

func cloneValue(value reflect.Value) (reflect.Value, error) {
	if !value.IsValid() {
		return reflect.New(reflect.TypeOf((*interface{})(nil)).Elem()).Elem(), nil
	}

	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		cloned, err := cloneValue(value.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		out := reflect.New(value.Type()).Elem()
		if cloned.IsValid() {
			out.Set(cloned)
		}
		return out, nil

	case reflect.Ptr:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		cloned, err := cloneValue(value.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		out := reflect.New(value.Elem().Type())
		out.Elem().Set(cloned)
		return out, nil

	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		out := reflect.MakeMapWithSize(value.Type(), value.Len())
		for _, key := range value.MapKeys() {
			clonedKey, err := cloneValue(key)
			if err != nil {
				return reflect.Value{}, err
			}
			clonedValue, err := cloneValue(value.MapIndex(key))
			if err != nil {
				return reflect.Value{}, err
			}
			out.SetMapIndex(clonedKey, clonedValue)
		}
		return out, nil

	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type()), nil
		}
		out := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			clonedItem, err := cloneValue(value.Index(i))
			if err != nil {
				return reflect.Value{}, err
			}
			out.Index(i).Set(clonedItem)
		}
		return out, nil

	case reflect.Array:
		out := reflect.New(value.Type()).Elem()
		for i := 0; i < value.Len(); i++ {
			clonedItem, err := cloneValue(value.Index(i))
			if err != nil {
				return reflect.Value{}, err
			}
			out.Index(i).Set(clonedItem)
		}
		return out, nil

	case reflect.Struct:
		out := reflect.New(value.Type()).Elem()
		for i := 0; i < value.NumField(); i++ {
			clonedField, err := cloneValue(value.Field(i))
			if err != nil {
				return reflect.Value{}, err
			}
			field := out.Field(i)
			if !field.CanSet() {
				continue
			}
			if !clonedField.IsValid() {
				field.Set(reflect.Zero(field.Type()))
				continue
			}
			if clonedField.Type().AssignableTo(field.Type()) {
				field.Set(clonedField)
				continue
			}
			if clonedField.Type().ConvertibleTo(field.Type()) {
				field.Set(clonedField.Convert(field.Type()))
			}
		}
		return out, nil

	default:
		return value, nil
	}
}
