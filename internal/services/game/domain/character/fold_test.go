package character

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestFoldCharacterCreatedSetsFields(t *testing.T) {
	state := State{}
	updated := Fold(state, event.Event{
		Type:        event.Type("character.created"),
		PayloadJSON: []byte(`{"character_id":"char-1","name":"Aria","kind":"pc","notes":"notes"}`),
	})
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
}

func TestFoldCharacterUpdatedSetsFields(t *testing.T) {
	state := State{Created: true, CharacterID: "char-1", Name: "Old"}
	updated := Fold(state, event.Event{
		Type:        event.Type("character.updated"),
		PayloadJSON: []byte(`{"character_id":"char-1","fields":{"name":"Aria","kind":"npc","notes":"new notes","participant_id":"p-1"}}`),
	})
	if updated.Name != "Aria" {
		t.Fatalf("name = %s, want %s", updated.Name, "Aria")
	}
	if updated.Kind != "npc" {
		t.Fatalf("kind = %s, want %s", updated.Kind, "npc")
	}
	if updated.Notes != "new notes" {
		t.Fatalf("notes = %s, want %s", updated.Notes, "new notes")
	}
	if updated.ParticipantID != "p-1" {
		t.Fatalf("participant id = %s, want %s", updated.ParticipantID, "p-1")
	}
}

func TestFoldCharacterDeletedMarksDeleted(t *testing.T) {
	state := State{Created: true, CharacterID: "char-1"}
	updated := Fold(state, event.Event{
		Type:        event.Type("character.deleted"),
		PayloadJSON: []byte(`{"character_id":"char-1"}`),
	})
	if !updated.Deleted {
		t.Fatal("expected character to be marked deleted")
	}
}

func TestFoldCharacterCreatedClearsDeleted(t *testing.T) {
	state := State{Deleted: true}
	updated := Fold(state, event.Event{
		Type:        event.Type("character.created"),
		PayloadJSON: []byte(`{"character_id":"char-1","name":"Aria","kind":"pc"}`),
	})
	if updated.Deleted {
		t.Fatal("expected deleted flag to be cleared")
	}
}

func TestFoldProfileUpdatedSetsSystemProfile(t *testing.T) {
	state := State{Created: true}
	updated := Fold(state, event.Event{
		Type:        event.Type("character.profile_updated"),
		PayloadJSON: []byte(`{"character_id":"char-1","system_profile":{"daggerheart":{"level":2}}}`),
	})
	if updated.SystemProfile == nil {
		t.Fatal("expected system profile to be set")
	}
	profile, ok := updated.SystemProfile["daggerheart"].(map[string]any)
	if !ok {
		t.Fatalf("expected daggerheart profile to be map, got %T", updated.SystemProfile["daggerheart"])
	}
	level, ok := profile["level"].(float64)
	if !ok {
		t.Fatalf("expected level to be float64, got %T", profile["level"])
	}
	if level != 2 {
		t.Fatalf("level = %v, want %v", level, 2)
	}
}
