package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestCharacterCRUD(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-char", now)
	seedParticipant(t, store, "camp-char", "part-1", "user-1", now)

	expected := character.Character{
		ID:            "char-1",
		CampaignID:    "camp-char",
		ParticipantID: "part-1",
		Name:          "Aria Starweaver",
		Kind:          character.CharacterKindPC,
		Notes:         "Brave adventurer",
		CreatedAt:     now,
		UpdatedAt:     now.Add(5 * time.Minute),
	}

	if err := store.PutCharacter(context.Background(), expected); err != nil {
		t.Fatalf("put character: %v", err)
	}

	got, err := store.GetCharacter(context.Background(), "camp-char", "char-1")
	if err != nil {
		t.Fatalf("get character: %v", err)
	}
	if got.ID != expected.ID || got.CampaignID != expected.CampaignID {
		t.Fatalf("expected character identity to match")
	}
	if got.ParticipantID != expected.ParticipantID {
		t.Fatalf("expected participant id %q, got %q", expected.ParticipantID, got.ParticipantID)
	}
	if got.Name != expected.Name {
		t.Fatalf("expected name %q, got %q", expected.Name, got.Name)
	}
	if got.Kind != expected.Kind {
		t.Fatalf("expected kind %v, got %v", expected.Kind, got.Kind)
	}
	if got.Notes != expected.Notes {
		t.Fatalf("expected notes to match")
	}
	if !got.CreatedAt.Equal(expected.CreatedAt) || !got.UpdatedAt.Equal(expected.UpdatedAt) {
		t.Fatalf("expected timestamps to match")
	}

	// Test with empty ParticipantID (NPC without owner)
	npc := character.Character{
		ID:         "char-npc",
		CampaignID: "camp-char",
		Name:       "Goblin",
		Kind:       character.CharacterKindNPC,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := store.PutCharacter(context.Background(), npc); err != nil {
		t.Fatalf("put NPC: %v", err)
	}
	gotNPC, err := store.GetCharacter(context.Background(), "camp-char", "char-npc")
	if err != nil {
		t.Fatalf("get NPC: %v", err)
	}
	if gotNPC.ParticipantID != "" {
		t.Fatalf("expected empty participant id for NPC, got %q", gotNPC.ParticipantID)
	}

	// Delete
	if err := store.DeleteCharacter(context.Background(), "camp-char", "char-1"); err != nil {
		t.Fatalf("delete character: %v", err)
	}
	_, err = store.GetCharacter(context.Background(), "camp-char", "char-1")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestCharacterGetNotFound(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-missing", now)

	_, err := store.GetCharacter(context.Background(), "camp-missing", "no-such-char")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCharacterListPaging(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-list", now)

	for _, id := range []string{"char-a", "char-b", "char-c"} {
		seedCharacter(t, store, "camp-list", id, "Name-"+id, character.CharacterKindPC, now)
	}

	page, err := store.ListCharacters(context.Background(), "camp-list", 2, "")
	if err != nil {
		t.Fatalf("list characters: %v", err)
	}
	if len(page.Characters) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(page.Characters))
	}
	if page.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	second, err := store.ListCharacters(context.Background(), "camp-list", 2, page.NextPageToken)
	if err != nil {
		t.Fatalf("list characters page 2: %v", err)
	}
	if len(second.Characters) != 1 {
		t.Fatalf("expected 1 character, got %d", len(second.Characters))
	}
	if second.NextPageToken != "" {
		t.Fatalf("expected empty next page token, got %s", second.NextPageToken)
	}
}
