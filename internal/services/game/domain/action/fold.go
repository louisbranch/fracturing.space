package action

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// FoldHandledTypes returns the event types handled by the action fold function.
func FoldHandledTypes() []event.Type {
	return []event.Type{
		EventTypeRollResolved,
		EventTypeOutcomeApplied,
	}
}

// Fold applies an event to action state. It returns an error if a recognized
// event carries a payload that cannot be unmarshalled.
func Fold(state State, evt event.Event) (State, error) {
	switch evt.Type {
	case EventTypeRollResolved:
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
	case EventTypeOutcomeApplied:
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
	}
	return state, nil
}
