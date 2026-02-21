package character

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// Fold applies an event to character state.
func Fold(state State, evt event.Event) State {
	if evt.Type == eventTypeCreated {
		state.Created = true
		state.Deleted = false
		var payload CreatePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		state.CharacterID = payload.CharacterID
		state.Name = payload.Name
		state.Kind = payload.Kind
		state.Notes = payload.Notes
		state.AvatarSetID = payload.AvatarSetID
		state.AvatarAssetID = payload.AvatarAssetID
	}
	if evt.Type == eventTypeUpdated {
		var payload UpdatePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.CharacterID != "" {
			state.CharacterID = payload.CharacterID
		}
		for key, value := range payload.Fields {
			switch key {
			case "name":
				state.Name = value
			case "kind":
				state.Kind = value
			case "notes":
				state.Notes = value
			case "participant_id":
				state.ParticipantID = value
			case "avatar_set_id":
				state.AvatarSetID = value
			case "avatar_asset_id":
				state.AvatarAssetID = value
			}
		}
	}
	if evt.Type == eventTypeDeleted {
		state.Deleted = true
		var payload DeletePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.CharacterID != "" {
			state.CharacterID = payload.CharacterID
		}
	}
	if evt.Type == eventTypeProfileUpdated {
		var payload ProfileUpdatePayload
		_ = json.Unmarshal(evt.PayloadJSON, &payload)
		if payload.CharacterID != "" {
			state.CharacterID = payload.CharacterID
		}
		state.SystemProfile = payload.SystemProfile
	}
	return state
}
