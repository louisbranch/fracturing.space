package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	"github.com/louisbranch/fracturing.space/internal/services/listing/storage"
)

func TestOpenRequiresPath(t *testing.T) {
	t.Parallel()

	if _, err := Open(""); err == nil {
		t.Fatal("expected empty path error")
	}
}

func TestCreateGetCampaignListingRoundTrip(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, time.February, 22, 16, 40, 0, 0, time.UTC)
	input := storage.CampaignListing{
		CampaignID:                 "camp-1",
		Title:                      "Sunfall",
		Description:                "A haunted valley campaign",
		RecommendedParticipantsMin: 3,
		RecommendedParticipantsMax: 5,
		DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER,
		ExpectedDurationLabel:      "2-3 sessions",
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	if err := store.CreateCampaignListing(context.Background(), input); err != nil {
		t.Fatalf("create campaign listing: %v", err)
	}

	got, err := store.GetCampaignListing(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("get campaign listing: %v", err)
	}
	if got.CampaignID != input.CampaignID {
		t.Fatalf("campaign_id = %q, want %q", got.CampaignID, input.CampaignID)
	}
	if got.Title != input.Title {
		t.Fatalf("title = %q, want %q", got.Title, input.Title)
	}
	if got.Description != input.Description {
		t.Fatalf("description = %q, want %q", got.Description, input.Description)
	}
}

func TestCreateCampaignListingReturnsAlreadyExistsOnDuplicate(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, time.February, 22, 16, 50, 0, 0, time.UTC)
	input := storage.CampaignListing{
		CampaignID:                 "camp-dup",
		Title:                      "Duplicate Campaign",
		Description:                "Starter listing",
		RecommendedParticipantsMin: 3,
		RecommendedParticipantsMax: 5,
		DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_INTERMEDIATE,
		ExpectedDurationLabel:      "3-4 sessions",
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}
	if err := store.CreateCampaignListing(context.Background(), input); err != nil {
		t.Fatalf("create initial listing: %v", err)
	}
	err := store.CreateCampaignListing(context.Background(), input)
	if !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("duplicate create error = %v, want %v", err, storage.ErrAlreadyExists)
	}
}

func TestListCampaignListingsPaginates(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, time.February, 22, 17, 0, 0, 0, time.UTC)
	for _, id := range []string{"camp-1", "camp-2", "camp-3"} {
		if err := store.CreateCampaignListing(context.Background(), storage.CampaignListing{
			CampaignID:                 id,
			Title:                      "Title " + id,
			Description:                "Desc " + id,
			RecommendedParticipantsMin: 2,
			RecommendedParticipantsMax: 4,
			DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER,
			ExpectedDurationLabel:      "1 session",
			CreatedAt:                  now,
			UpdatedAt:                  now,
		}); err != nil {
			t.Fatalf("create listing %s: %v", id, err)
		}
	}

	pageOne, err := store.ListCampaignListings(context.Background(), 2, "")
	if err != nil {
		t.Fatalf("list page one: %v", err)
	}
	if len(pageOne.Listings) != 2 {
		t.Fatalf("page one len = %d, want 2", len(pageOne.Listings))
	}
	if pageOne.NextPageToken == "" {
		t.Fatal("expected page one next token")
	}

	pageTwo, err := store.ListCampaignListings(context.Background(), 2, pageOne.NextPageToken)
	if err != nil {
		t.Fatalf("list page two: %v", err)
	}
	if len(pageTwo.Listings) != 1 {
		t.Fatalf("page two len = %d, want 1", len(pageTwo.Listings))
	}
	if pageTwo.NextPageToken != "" {
		t.Fatalf("page two next token = %q, want empty", pageTwo.NextPageToken)
	}
}

func TestCampaignListingsSchemaRejectsInvalidParticipantBounds(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, time.February, 22, 17, 10, 0, 0, time.UTC).UnixMilli()
	testCases := []struct {
		name string
		min  int
		max  int
	}{
		{
			name: "min must be positive",
			min:  0,
			max:  4,
		},
		{
			name: "max must be greater than or equal to min",
			min:  5,
			max:  4,
		},
	}

	for idx, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := store.sqlDB.ExecContext(
				context.Background(),
				`INSERT INTO campaign_listings (
				   campaign_id,
				   title,
				   description,
				   recommended_participants_min,
				   recommended_participants_max,
				   difficulty_tier,
				   expected_duration_label,
				   system,
				   created_at,
				   updated_at
				 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				"invalid-camp-"+string(rune('a'+idx)),
				"Broken listing",
				"Used for schema validation",
				tc.min,
				tc.max,
				int32(listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER),
				"2 sessions",
				0,
				now,
				now,
			)
			if err == nil {
				t.Fatal("expected schema constraint error")
			}
		})
	}
}

func TestIsCampaignListingUniqueViolation_DoesNotTreatCheckConstraintAsUnique(t *testing.T) {
	t.Parallel()

	store := openTempStore(t)
	now := time.Date(2026, time.February, 22, 17, 20, 0, 0, time.UTC).UnixMilli()
	_, err := store.sqlDB.ExecContext(
		context.Background(),
		`INSERT INTO campaign_listings (
		   campaign_id,
		   title,
		   description,
		   recommended_participants_min,
		   recommended_participants_max,
		   difficulty_tier,
		   expected_duration_label,
		   system,
		   created_at,
		   updated_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"check-constraint-camp",
		"Broken listing",
		"Used for unique violation classification",
		0,
		4,
		int32(listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER),
		"2 sessions",
		0,
		now,
		now,
	)
	if err == nil {
		t.Fatal("expected constraint error")
	}
	if isCampaignListingUniqueViolation(err) {
		t.Fatalf("check constraint error incorrectly classified as unique violation: %v", err)
	}
}

func openTempStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(filepath.Join(t.TempDir(), "listing.db"))
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
