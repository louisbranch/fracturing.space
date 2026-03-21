package action

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/fold"
)

// foldRouter is the registration-based fold dispatcher. Handled types are
// derived from registered handlers, eliminating sync-drift between the switch
// and the type list.
var foldRouter = newFoldRouter()

func newFoldRouter() *fold.CoreFoldRouter[State] {
	r := fold.NewCoreFoldRouter[State]()
	r.Handle(EventTypeRollResolved, foldRollResolved)
	r.Handle(EventTypeOutcomeApplied, foldOutcomeApplied)
	return r
}

// FoldHandledTypes returns the event types handled by the action fold function.
// Derived from registered handlers via the fold router.
func FoldHandledTypes() []event.Type {
	return foldRouter.FoldHandledTypes()
}

// Fold applies an event to action state. Returns an error for unhandled
// event types and for recognized events with unparseable payloads.
func Fold(state State, evt event.Event) (State, error) {
	return foldRouter.Fold(state, evt)
}

func foldRollResolved(state State, evt event.Event) (State, error) {
	var payload RollResolvePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("action fold %s: %w", evt.Type, err)
	}
	if payload.RollSeq == 0 {
		return state, nil
	}
	if state.Rolls == nil {
		state.Rolls = make(map[uint64]RollState)
	}
	state.Rolls[payload.RollSeq] = RollState{
		RequestID: payload.RequestID,
		SessionID: evt.SessionID,
		Outcome:   payload.Outcome,
	}
	return state, nil
}

func foldOutcomeApplied(state State, evt event.Event) (State, error) {
	var payload OutcomeApplyPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("action fold %s: %w", evt.Type, err)
	}
	if payload.RollSeq == 0 {
		return state, nil
	}
	if state.AppliedOutcomes == nil {
		state.AppliedOutcomes = make(map[uint64]struct{})
	}
	state.AppliedOutcomes[payload.RollSeq] = struct{}{}
	return state, nil
}
