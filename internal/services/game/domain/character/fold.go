package character

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// FoldHandledTypes returns the event types handled by the character fold function.
func FoldHandledTypes() []event.Type {
	return []event.Type{
		EventTypeCreated,
		EventTypeUpdated,
		EventTypeDeleted,
		EventTypeProfileUpdated,
	}
}

// Fold applies an event to character state. It returns an error if a recognized
// event carries a payload that cannot be unmarshalled.
func Fold(state State, evt event.Event) (State, error) {
	switch evt.Type {
	case EventTypeCreated:
		state.Created = true
		state.Deleted = false
		var payload CreatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("character fold %s: %w", evt.Type, err)
		}
		state.CharacterID = payload.CharacterID
		state.OwnerParticipantID = payload.OwnerParticipantID
		state.ParticipantID = strings.TrimSpace(payload.ParticipantID)
		state.Name = payload.Name
		state.Kind = payload.Kind
		state.Notes = payload.Notes
		state.AvatarSetID = payload.AvatarSetID
		state.AvatarAssetID = payload.AvatarAssetID
		state.Pronouns = payload.Pronouns
		state.Aliases = normalizeAliases(payload.Aliases)
	case EventTypeUpdated:
		var payload UpdatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("character fold %s: %w", evt.Type, err)
		}
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
			case "owner_participant_id":
				state.OwnerParticipantID = value
			case "avatar_set_id":
				state.AvatarSetID = value
			case "avatar_asset_id":
				state.AvatarAssetID = value
			case "pronouns":
				state.Pronouns = value
			case "aliases":
				aliases, err := normalizeAliasesField(value)
				if err != nil {
					return state, fmt.Errorf("character fold %s: %w", evt.Type, err)
				}
				state.Aliases = aliases
			}
		}
	case EventTypeDeleted:
		state.Deleted = true
		var payload DeletePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("character fold %s: %w", evt.Type, err)
		}
		if payload.CharacterID != "" {
			state.CharacterID = payload.CharacterID
		}
	case EventTypeProfileUpdated:
		var payload ProfileUpdatePayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return state, fmt.Errorf("character fold %s: %w", evt.Type, err)
		}
		if payload.CharacterID != "" {
			state.CharacterID = payload.CharacterID
		}
		state.SystemProfile = payload.SystemProfile
	}
	return state, nil
}
