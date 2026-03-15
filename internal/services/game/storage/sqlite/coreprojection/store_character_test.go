package coreprojection

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestCharacterCRUD(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-char", now)
	seedParticipant(t, store, "camp-char", "part-1", "user-1", now)

	expected := storage.CharacterRecord{
		ID:            "char-1",
		CampaignID:    "camp-char",
		ParticipantID: "part-1",
		Name:          "Aria Starweaver",
		Kind:          character.KindPC,
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
	npc := storage.CharacterRecord{
		ID:         "char-npc",
		CampaignID: "camp-char",
		Name:       "Goblin",
		Kind:       character.KindNPC,
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
		seedCharacter(t, store, "camp-list", id, "Name-"+id, character.KindPC, now)
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

func TestCharacterListByOwnerParticipant(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-owned", now)
	seedParticipant(t, store, "camp-owned", "part-1", "user-1", now)
	seedParticipant(t, store, "camp-owned", "part-2", "user-2", now)

	if err := store.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:                 "char-b",
		CampaignID:         "camp-owned",
		OwnerParticipantID: "part-1",
		Name:               "Second",
		Kind:               character.KindPC,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatalf("put character b: %v", err)
	}
	if err := store.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:                 "char-a",
		CampaignID:         "camp-owned",
		OwnerParticipantID: "part-1",
		Name:               "First",
		Kind:               character.KindPC,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatalf("put character a: %v", err)
	}
	if err := store.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:                 "char-c",
		CampaignID:         "camp-owned",
		OwnerParticipantID: "part-2",
		Name:               "Other",
		Kind:               character.KindPC,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatalf("put character c: %v", err)
	}

	characters, err := store.ListCharactersByOwnerParticipant(context.Background(), "camp-owned", "part-1")
	if err != nil {
		t.Fatalf("list characters by owner: %v", err)
	}
	if len(characters) != 2 {
		t.Fatalf("expected 2 owned characters, got %d", len(characters))
	}
	if characters[0].ID != "char-a" || characters[1].ID != "char-b" {
		t.Fatalf("unexpected owner character order: %#v", characters)
	}
}

func TestCharacterListByControllerParticipant(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-controlled", now)
	seedParticipant(t, store, "camp-controlled", "part-1", "user-1", now)
	seedParticipant(t, store, "camp-controlled", "part-2", "user-2", now)

	if err := store.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:                 "char-b",
		CampaignID:         "camp-controlled",
		OwnerParticipantID: "owner-1",
		ParticipantID:      "part-1",
		Name:               "Second",
		Kind:               character.KindPC,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatalf("put character b: %v", err)
	}
	if err := store.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:                 "char-a",
		CampaignID:         "camp-controlled",
		OwnerParticipantID: "owner-2",
		ParticipantID:      "part-1",
		Name:               "First",
		Kind:               character.KindPC,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatalf("put character a: %v", err)
	}
	if err := store.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:                 "char-c",
		CampaignID:         "camp-controlled",
		OwnerParticipantID: "owner-3",
		ParticipantID:      "part-2",
		Name:               "Other",
		Kind:               character.KindPC,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatalf("put character c: %v", err)
	}

	characters, err := store.ListCharactersByControllerParticipant(context.Background(), "camp-controlled", "part-1")
	if err != nil {
		t.Fatalf("list characters by controller: %v", err)
	}
	if len(characters) != 2 {
		t.Fatalf("expected 2 controlled characters, got %d", len(characters))
	}
	if characters[0].ID != "char-a" || characters[1].ID != "char-b" {
		t.Fatalf("unexpected controlled character order: %#v", characters)
	}
}
