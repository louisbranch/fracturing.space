package session

import (
	"encoding/json"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// FoldHandledTypes returns the event types handled by the session fold function.
func FoldHandledTypes() []event.Type {
	return []event.Type{
		EventTypeStarted,
		EventTypeEnded,
		EventTypeGateOpened,
		EventTypeGateResolved,
		EventTypeGateAbandoned,
		EventTypeSpotlightSet,
		EventTypeSpotlightCleared,
	}
}

// Fold applies an event to session state. It returns an error if a recognized
// event carries a payload that cannot be unmarshalled.
//
// The fold is intentionally declarative: every session transition is represented as
// an event so tests and replay both observe the same gate and spotlight behavior.
func Fold(state State, evt event.Event) (State, error) {
	switch evt.Type {
	case EventTypeStarted:
		state.Started = true
		state.Ended = false
		var payload StartPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		state.SessionID = payload.SessionID
		state.Name = payload.SessionName
	case EventTypeEnded:
		state.Ended = true
		state.Started = false
		var payload EndPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		if payload.SessionID != "" {
			state.SessionID = payload.SessionID
		}
	case EventTypeGateOpened:
		state.GateOpen = true
		var payload GateOpenedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		state.GateID = payload.GateID
	case EventTypeGateResolved, EventTypeGateAbandoned:
		state.GateOpen = false
		state.GateID = ""
	case EventTypeSpotlightSet:
		var payload SpotlightSetPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("session fold %s: %w", evt.Type, err)
		}
		state.SpotlightType = payload.SpotlightType
		state.SpotlightCharacterID = payload.CharacterID
	case EventTypeSpotlightCleared:
		state.SpotlightType = ""
		state.SpotlightCharacterID = ""
	}
	return state, nil
}
