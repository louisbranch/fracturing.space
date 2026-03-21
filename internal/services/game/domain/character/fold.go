package character

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/fold"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

type characterUpdateFieldApplier func(*State, string) error

var characterUpdateFieldAppliers = map[string]characterUpdateFieldApplier{
	"name": func(state *State, value string) error {
		state.Name = value
		return nil
	},
	"kind": func(state *State, value string) error {
		state.Kind = Kind(value)
		return nil
	},
	"notes": func(state *State, value string) error {
		state.Notes = value
		return nil
	},
	"participant_id": func(state *State, value string) error {
		state.ParticipantID = ids.ParticipantID(value)
		return nil
	},
	"owner_participant_id": func(state *State, value string) error {
		state.OwnerParticipantID = ids.ParticipantID(value)
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

// foldRouter is the registration-based fold dispatcher. Handled types are
// derived from registered handlers, eliminating sync-drift between the switch
// and the type list.
var foldRouter = newFoldRouter()

func newFoldRouter() *fold.CoreFoldRouter[State] {
	r := fold.NewCoreFoldRouter[State]()
	r.Handle(EventTypeCreated, foldCreated)
	r.Handle(EventTypeUpdated, foldUpdated)
	r.Handle(EventTypeDeleted, foldDeleted)
	return r
}

// FoldHandledTypes returns the event types handled by the character fold function.
// Derived from registered handlers via the fold router.
func FoldHandledTypes() []event.Type {
	return foldRouter.FoldHandledTypes()
}

// Fold applies an event to character state. Returns an error for unhandled
// event types and for recognized events with unparseable payloads.
func Fold(state State, evt event.Event) (State, error) {
	return foldRouter.Fold(state, evt)
}

func foldCreated(state State, evt event.Event) (State, error) {
	state.Created = true
	state.Deleted = false

	var payload CreatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		return state, fmt.Errorf("character fold %s: %w", evt.Type, err)
	}
	state.CharacterID = ids.CharacterID(payload.CharacterID)
	state.OwnerParticipantID = ids.ParticipantID(payload.OwnerParticipantID)
	state.ParticipantID = ids.ParticipantID(strings.TrimSpace(payload.ParticipantID.String()))
	state.Name = payload.Name
	state.Kind = Kind(payload.Kind)
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
		state.CharacterID = ids.CharacterID(payload.CharacterID)
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
		state.CharacterID = ids.CharacterID(payload.CharacterID)
	}
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
