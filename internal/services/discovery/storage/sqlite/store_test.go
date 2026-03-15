package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
)

func TestOpenRequiresPath(t *testing.T) {
	t.Parallel()
	if _, err := Open(""); err == nil {
		t.Fatal("expected empty path error")
	}
}

func TestCreateGetDiscoveryEntryRoundTrip(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, time.March, 6, 13, 0, 0, 0, time.UTC)
	input := storage.DiscoveryEntry{
		EntryID:                    "starter:camp-1",
		Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
		SourceID:                   "",
		Title:                      "Sunfall",
		Description:                "A haunted valley campaign",
		CampaignTheme:              "A village lighthouse has gone dark.\nYou must restore it before the next fleet arrives.",
		RecommendedParticipantsMin: 3,
		RecommendedParticipantsMax: 5,
		DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "2-3 sessions",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:                     discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI,
		Intent:                     discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_STARTER,
		Level:                      1,
		CharacterCount:             1,
		Storyline:                  "# Test Storyline",
		Tags:                       []string{"solo", "mystery"},
		PreviewHook:                "A dark bell tolls across the valley.",
		PreviewPlaystyleLabel:      "Guardian defender",
		PreviewCharacterName:       "Mira Vale",
		PreviewCharacterSummary:    "A steadfast guardian.",
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	if err := store.CreateDiscoveryEntry(context.Background(), input); err != nil {
		t.Fatalf("create discovery entry: %v", err)
	}

	got, err := store.GetDiscoveryEntry(context.Background(), input.EntryID)
	if err != nil {
		t.Fatalf("get discovery entry: %v", err)
	}
	if got.EntryID != input.EntryID || got.SourceID != input.SourceID {
		t.Fatalf("got ids (%q,%q), want (%q,%q)", got.EntryID, got.SourceID, input.EntryID, input.SourceID)
	}
	if got.Kind != input.Kind {
		t.Fatalf("kind = %v, want %v", got.Kind, input.Kind)
	}
	if got.GmMode != input.GmMode || got.Intent != input.Intent {
		t.Fatalf("gm/intent mismatch: got (%v,%v), want (%v,%v)", got.GmMode, got.Intent, input.GmMode, input.Intent)
	}
	if got.PreviewCharacterName != input.PreviewCharacterName || got.PreviewHook != input.PreviewHook {
		t.Fatalf("preview fields mismatch: got (%q,%q)", got.PreviewCharacterName, got.PreviewHook)
	}
	if got.CampaignTheme != input.CampaignTheme {
		t.Fatalf("campaign theme = %q, want %q", got.CampaignTheme, input.CampaignTheme)
	}
}

func TestCreateDiscoveryEntryReturnsAlreadyExistsOnDuplicate(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	input := storage.DiscoveryEntry{
		EntryID:                    "starter:dup",
		Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
		SourceID:                   "",
		Title:                      "Duplicate",
		Description:                "Duplicate",
		RecommendedParticipantsMin: 2,
		RecommendedParticipantsMax: 4,
		DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "1 session",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}
	if err := store.CreateDiscoveryEntry(context.Background(), input); err != nil {
		t.Fatalf("create initial entry: %v", err)
	}
	err := store.CreateDiscoveryEntry(context.Background(), input)
	if !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("duplicate error = %v, want %v", err, storage.ErrAlreadyExists)
	}
}

func TestListDiscoveryEntriesPaginatesAndFiltersByKind(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, time.March, 6, 13, 10, 0, 0, time.UTC)
	entries := []storage.DiscoveryEntry{
		{
			EntryID:                    "entry-1",
			Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
			SourceID:                   "",
			Title:                      "Starter 1",
			Description:                "Starter 1",
			RecommendedParticipantsMin: 1,
			RecommendedParticipantsMax: 1,
			DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
			ExpectedDurationLabel:      "1 session",
			System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			CreatedAt:                  now,
			UpdatedAt:                  now,
		},
		{
			EntryID:                    "entry-2",
			Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
			SourceID:                   "",
			Title:                      "Starter 2",
			Description:                "Starter 2",
			RecommendedParticipantsMin: 1,
			RecommendedParticipantsMax: 1,
			DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
			ExpectedDurationLabel:      "1 session",
			System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			CreatedAt:                  now,
			UpdatedAt:                  now,
		},
		{
			EntryID:                    "entry-3",
			Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_STORYLINE,
			SourceID:                   "",
			Title:                      "Storyline",
			Description:                "Storyline",
			RecommendedParticipantsMin: 1,
			RecommendedParticipantsMax: 1,
			DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
			ExpectedDurationLabel:      "1 session",
			System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			CreatedAt:                  now,
			UpdatedAt:                  now,
		},
	}
	for _, entry := range entries {
		if err := store.CreateDiscoveryEntry(context.Background(), entry); err != nil {
			t.Fatalf("create %s: %v", entry.EntryID, err)
		}
	}

	pageOne, err := store.ListDiscoveryEntries(context.Background(), 1, "", discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER)
	if err != nil {
		t.Fatalf("list page one: %v", err)
	}
	if len(pageOne.Entries) != 1 || pageOne.NextPageToken == "" {
		t.Fatalf("page one invalid: %+v", pageOne)
	}

	pageTwo, err := store.ListDiscoveryEntries(context.Background(), 1, pageOne.NextPageToken, discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER)
	if err != nil {
		t.Fatalf("list page two: %v", err)
	}
	if len(pageTwo.Entries) != 1 {
		t.Fatalf("page two len = %d, want 1", len(pageTwo.Entries))
	}
	if pageTwo.Entries[0].Kind != discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER {
		t.Fatalf("page two kind = %v, want CAMPAIGN_STARTER", pageTwo.Entries[0].Kind)
	}
}

func TestUpsertBuiltinDiscoveryEntryPreservesReconciledSourceID(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, time.March, 6, 14, 0, 0, 0, time.UTC)
	initial := storage.DiscoveryEntry{
		EntryID:                    "starter:camp-1",
		Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
		SourceID:                   "camp-1",
		Title:                      "Sunfall",
		Description:                "A haunted valley campaign",
		CampaignTheme:              "Initial theme",
		RecommendedParticipantsMin: 1,
		RecommendedParticipantsMax: 1,
		DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "1 session",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	if err := store.UpsertBuiltinDiscoveryEntry(context.Background(), initial); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	updated := initial
	updated.SourceID = ""
	updated.Description = "Updated description"
	updated.CampaignTheme = "Updated theme"
	updated.PreviewCharacterName = "Mira Vale"
	if err := store.UpsertBuiltinDiscoveryEntry(context.Background(), updated); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	got, err := store.GetDiscoveryEntry(context.Background(), initial.EntryID)
	if err != nil {
		t.Fatalf("get discovery entry: %v", err)
	}
	if got.SourceID != "camp-1" {
		t.Fatalf("source_id = %q, want camp-1", got.SourceID)
	}
	if got.Description != "Updated description" {
		t.Fatalf("description = %q, want updated", got.Description)
	}
	if got.CampaignTheme != "Updated theme" {
		t.Fatalf("campaign_theme = %q, want updated", got.CampaignTheme)
	}
}

func TestUpdateDiscoveryEntrySourceIDUpdatesSourceID(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, time.March, 6, 15, 0, 0, 0, time.UTC)
	entry := storage.DiscoveryEntry{
		EntryID:                    "starter:camp-1",
		Kind:                       discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
		Title:                      "Sunfall",
		Description:                "A haunted valley campaign",
		RecommendedParticipantsMin: 1,
		RecommendedParticipantsMax: 1,
		DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "1 session",
		System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	if err := store.CreateDiscoveryEntry(context.Background(), entry); err != nil {
		t.Fatalf("create discovery entry: %v", err)
	}
	if err := store.UpdateDiscoveryEntrySourceID(context.Background(), entry.EntryID, "camp-1", now.Add(time.Minute)); err != nil {
		t.Fatalf("update source id: %v", err)
	}

	got, err := store.GetDiscoveryEntry(context.Background(), entry.EntryID)
	if err != nil {
		t.Fatalf("get discovery entry: %v", err)
	}
	if got.SourceID != "camp-1" {
		t.Fatalf("source_id = %q, want camp-1", got.SourceID)
	}
}

func openTempStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(filepath.Join(t.TempDir(), "discovery.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	return store
}
