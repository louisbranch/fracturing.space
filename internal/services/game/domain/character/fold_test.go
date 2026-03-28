package character

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestFoldCharacterCreatedSetsFields(t *testing.T) {
	state := State{}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("character.created"),
		PayloadJSON: []byte(`{"character_id":"char-1","owner_participant_id":"p-owner","name":"Aria","kind":"pc","notes":"notes"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.Created {
		t.Fatal("expected character to be created")
	}
	if updated.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", updated.CharacterID, "char-1")
	}
	if updated.Name != "Aria" {
		t.Fatalf("name = %s, want %s", updated.Name, "Aria")
	}
	if updated.Kind != "pc" {
		t.Fatalf("kind = %s, want %s", updated.Kind, "pc")
	}
	if updated.Notes != "notes" {
		t.Fatalf("notes = %s, want %s", updated.Notes, "notes")
	}
	if updated.OwnerParticipantID != "p-owner" {
		t.Fatalf("owner participant id = %s, want %s", updated.OwnerParticipantID, "p-owner")
	}
}

func TestFoldCharacterUpdatedSetsFields(t *testing.T) {
	state := State{Created: true, CharacterID: "char-1", Name: "Old"}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("character.updated"),
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"name":"Aria","kind":"npc","notes":"new notes"}}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Aria" {
		t.Fatalf("name = %s, want %s", updated.Name, "Aria")
	}
	if updated.Kind != "npc" {
		t.Fatalf("kind = %s, want %s", updated.Kind, "npc")
	}
	if updated.Notes != "new notes" {
		t.Fatalf("notes = %s, want %s", updated.Notes, "new notes")
	}
}

func TestFoldCharacterUpdatedSetsOwnerParticipantID(t *testing.T) {
	state := State{Created: true, CharacterID: "char-1", OwnerParticipantID: "p-1"}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("character.updated"),
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"owner_participant_id":"p-2"}}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.OwnerParticipantID != "p-2" {
		t.Fatalf("owner participant id = %s, want %s", updated.OwnerParticipantID, "p-2")
	}
}

func TestFoldCharacterDeletedMarksDeleted(t *testing.T) {
	state := State{Created: true, CharacterID: "char-1"}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("character.deleted"),
		PayloadJSON: []byte(`{"character_id":"char-1"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.Deleted {
		t.Fatal("expected character to be marked deleted")
	}
}

func TestFoldCharacterCreatedClearsDeleted(t *testing.T) {
	state := State{Deleted: true}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("character.created"),
		PayloadJSON: []byte(`{"character_id":"char-1","name":"Aria","kind":"pc"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Deleted {
		t.Fatal("expected deleted flag to be cleared")
	}
}

func TestFoldCharacterRecognizedEvents_InvalidPayloadReturnsError(t *testing.T) {
	eventTypes := []event.Type{
		EventTypeCreated,
		EventTypeUpdated,
		EventTypeDeleted,
	}

	for _, eventType := range eventTypes {
		t.Run(string(eventType), func(t *testing.T) {
			_, err := Fold(State{}, event.Event{
				Type:        eventType,
				PayloadJSON: []byte(`{bad json`),
			})
			if err == nil {
				t.Fatal("expected error for invalid payload")
			}
		})
	}
}

func TestFoldCharacterUpdated_AppliesAvatarPronounsAndAliasesFields(t *testing.T) {
	state := State{Created: true, CharacterID: "char-1"}
	updated, err := Fold(state, event.Event{
		Type: event.Type("character.updated"),
		PayloadJSON: []byte(
			`{"character_id":"char-1","fields":{"avatar_set_id":"set-2","avatar_asset_id":"asset-2","pronouns":"she/her","aliases":"[\"Aria\",\"Aria\",\"  Storm  \"]"}}`,
		),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.AvatarSetID != "set-2" {
		t.Fatalf("avatar set = %q, want %q", updated.AvatarSetID, "set-2")
	}
	if updated.AvatarAssetID != "asset-2" {
		t.Fatalf("avatar asset = %q, want %q", updated.AvatarAssetID, "asset-2")
	}
	if updated.Pronouns != "she/her" {
		t.Fatalf("pronouns = %q, want %q", updated.Pronouns, "she/her")
	}
	if len(updated.Aliases) != 2 || updated.Aliases[0] != "Aria" || updated.Aliases[1] != "Storm" {
		t.Fatalf("aliases = %v, want [Aria Storm]", updated.Aliases)
	}
}

func TestFoldCharacterUpdated_InvalidAliasesFieldReturnsError(t *testing.T) {
	_, err := Fold(State{Created: true}, event.Event{
		Type:        event.Type("character.updated"),
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"aliases":"not-json"}}`),
	})
	if err == nil {
		t.Fatal("expected error for invalid aliases payload")
	}
}
