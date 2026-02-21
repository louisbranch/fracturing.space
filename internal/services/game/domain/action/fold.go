package action

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Fold applies an event to action state.
func Fold(state State, evt event.Event) State {
	switch evt.Type {
	case eventTypeRollResolved:
		var payload RollResolvePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.RollSeq == 0 {
			return state
		}
		if state.Rolls == nil {
			state.Rolls = make(map[uint64]RollState)
		}
		state.Rolls[payload.RollSeq] = RollState{
			RequestID: payload.RequestID,
			SessionID: evt.SessionID,
			Outcome:   payload.Outcome,
		}
	case eventTypeOutcomeApplied:
		var payload OutcomeApplyPayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.RollSeq == 0 {
			return state
		}
		if state.AppliedOutcomes == nil {
			state.AppliedOutcomes = make(map[uint64]struct{})
		}
		state.AppliedOutcomes[payload.RollSeq] = struct{}{}
	}
	return state
}
