package session

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Fold applies an event to session state.
func Fold(state State, evt event.Event) State {
	if evt.Type == eventTypeStarted {
		state.Started = true
		state.Ended = false
		var payload StartPayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		state.SessionID = payload.SessionID
		state.Name = payload.SessionName
	}
	if evt.Type == eventTypeEnded {
		state.Ended = true
		state.Started = false
		var payload EndPayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.SessionID != "" {
			state.SessionID = payload.SessionID
		}
	}
	if evt.Type == eventTypeGateOpened {
		state.GateOpen = true
		var payload GateOpenedPayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		state.GateID = payload.GateID
	}
	if evt.Type == eventTypeGateResolved || evt.Type == eventTypeGateAbandoned {
		state.GateOpen = false
		state.GateID = ""
	}
	if evt.Type == eventTypeSpotlightSet {
		var payload SpotlightSetPayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		state.SpotlightType = payload.SpotlightType
		state.SpotlightCharacterID = payload.CharacterID
	}
	if evt.Type == eventTypeSpotlightCleared {
		state.SpotlightType = ""
		state.SpotlightCharacterID = ""
	}
	return state
}
