package character

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type characterUpdateFieldApplier func(*State, string) error

var characterUpdateFieldAppliers = map[string]characterUpdateFieldApplier{
	"name": func(state *State, value string) error {
		state.Name = value
		return nil
	},
	"kind": func(state *State, value string) error {
		state.Kind = value
		return nil
	},
	"notes": func(state *State, value string) error {
		state.Notes = value
		return nil
	},
	"participant_id": func(state *State, value string) error {
		state.ParticipantID = value
		return nil
	},
	"owner_participant_id": func(state *State, value string) error {
		state.OwnerParticipantID = value
		return nil
	},
	"avatar_set_id": func(state *State, value string) error {
		state.AvatarSetID = value
		return nil
	},
	"avatar_asset_id": func(state *State, value string) error {
		state.AvatarAssetID = value
		return nil
	},
	"pronouns": func(state *State, value string) error {
		state.Pronouns = value
		return nil
	},
	"aliases": func(state *State, value string) error {
		aliases, err := normalizeAliasesField(value)
		if err != nil {
			return err
		}
		state.Aliases = aliases
		return nil
	},
}

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
		return foldCreated(state, evt)
	case EventTypeUpdated:
		return foldUpdated(state, evt)
	case EventTypeDeleted:
		return foldDeleted(state, evt)
	case EventTypeProfileUpdated:
		return foldProfileUpdated(state, evt)
	}
	return state, nil
}

func foldCreated(state State, evt event.Event) (State, error) {
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
	return state, nil
}

func foldUpdated(state State, evt event.Event) (State, error) {
	var payload UpdatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("character fold %s: %w", evt.Type, err)
	}
	if payload.CharacterID != "" {
		state.CharacterID = payload.CharacterID
	}
	if err := applyCharacterUpdateFields(&state, payload.Fields); err != nil {
		return state, fmt.Errorf("character fold %s: %w", evt.Type, err)
	}
	return state, nil
}

func foldDeleted(state State, evt event.Event) (State, error) {
	state.Deleted = true

	var payload DeletePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("character fold %s: %w", evt.Type, err)
	}
	if payload.CharacterID != "" {
		state.CharacterID = payload.CharacterID
	}
	return state, nil
}

func foldProfileUpdated(state State, evt event.Event) (State, error) {
	var payload ProfileUpdatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("character fold %s: %w", evt.Type, err)
	}
	if payload.CharacterID != "" {
		state.CharacterID = payload.CharacterID
	}
	state.SystemProfile = payload.SystemProfile
	return state, nil
}

func applyCharacterUpdateFields(state *State, fields map[string]string) error {
	for key, value := range fields {
		applier, ok := characterUpdateFieldAppliers[key]
		if !ok {
			continue
		}
		if err := applier(state, value); err != nil {
			return err
		}
	}
	return nil
}
