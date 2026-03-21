package scenetransport

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestSceneToProto_WithCharacters(t *testing.T) {
	now := time.Now()
	ended := now.Add(time.Hour)
	rec := storage.SceneRecord{
		SceneID:     "sc-1",
		SessionID:   "s-1",
		Name:        "Battle",
		Description: "A fierce battle",
		Open:        false,
		CreatedAt:   now,
		UpdatedAt:   now,
		EndedAt:     &ended,
	}
	chars := []storage.SceneCharacterRecord{
		{CharacterID: "char-1"},
		{CharacterID: "char-2"},
	}
	pb := SceneToProto(rec, chars)
	if pb.GetName() != "Battle" {
		t.Errorf("name = %q, want %q", pb.GetName(), "Battle")
	}
	if len(pb.GetCharacterIds()) != 2 {
		t.Errorf("character count = %d, want 2", len(pb.GetCharacterIds()))
	}
	if pb.GetEndedAt() == nil {
		t.Error("expected ended_at")
	}
}

func TestSceneToProto_NoCharacters(t *testing.T) {
	rec := storage.SceneRecord{
		SceneID:   "sc-1",
		SessionID: "s-1",
		Name:      "Tavern",
		Open:      true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	pb := SceneToProto(rec, nil)
	if len(pb.GetCharacterIds()) != 0 {
		t.Errorf("character_ids = %v, want empty", pb.GetCharacterIds())
	}
	if pb.GetEndedAt() != nil {
		t.Error("expected nil ended_at")
	}
}
