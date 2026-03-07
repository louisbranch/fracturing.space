package catalog

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
)

func TestBuiltinListings_ReturnsThreeEntries(t *testing.T) {
	listings, err := BuiltinListings()
	if err != nil {
		t.Fatalf("BuiltinListings: %v", err)
	}
	if len(listings) != 3 {
		t.Fatalf("len(listings) = %d, want 3", len(listings))
	}
}

func TestBuiltinListings_DeterministicIDs(t *testing.T) {
	listings, err := BuiltinListings()
	if err != nil {
		t.Fatalf("BuiltinListings: %v", err)
	}
	wantIDs := []string{
		"starter:lantern-in-the-dark",
		"starter:whispers-of-the-thornwood",
		"starter:merchants-gambit",
	}
	for i, want := range wantIDs {
		if listings[i].CampaignID != want {
			t.Errorf("listings[%d].CampaignID = %q, want %q", i, listings[i].CampaignID, want)
		}
	}
}

func TestBuiltinListings_StorylinesNonEmpty(t *testing.T) {
	listings, err := BuiltinListings()
	if err != nil {
		t.Fatalf("BuiltinListings: %v", err)
	}
	for i, l := range listings {
		if l.Storyline == "" {
			t.Errorf("listings[%d] %q has empty storyline", i, l.CampaignID)
		}
		if len(l.Storyline) < 100 {
			t.Errorf("listings[%d] %q storyline suspiciously short (%d bytes)", i, l.CampaignID, len(l.Storyline))
		}
	}
}

func TestBuiltinListings_FieldValues(t *testing.T) {
	listings, err := BuiltinListings()
	if err != nil {
		t.Fatalf("BuiltinListings: %v", err)
	}
	for i, l := range listings {
		if l.Title == "" {
			t.Errorf("listings[%d] has empty title", i)
		}
		if l.Description == "" {
			t.Errorf("listings[%d] has empty description", i)
		}
		if l.RecommendedParticipantsMin != 1 {
			t.Errorf("listings[%d] min participants = %d, want 1", i, l.RecommendedParticipantsMin)
		}
		if l.RecommendedParticipantsMax != 1 {
			t.Errorf("listings[%d] max participants = %d, want 1", i, l.RecommendedParticipantsMax)
		}
		if l.DifficultyTier != listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER {
			t.Errorf("listings[%d] difficulty = %v, want BEGINNER", i, l.DifficultyTier)
		}
		if l.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
			t.Errorf("listings[%d] system = %v, want DAGGERHEART", i, l.System)
		}
		if l.GmMode != listingv1.CampaignListingGmMode_CAMPAIGN_LISTING_GM_MODE_AI {
			t.Errorf("listings[%d] gm_mode = %v, want AI", i, l.GmMode)
		}
		if l.Intent != listingv1.CampaignListingIntent_CAMPAIGN_LISTING_INTENT_STARTER {
			t.Errorf("listings[%d] intent = %v, want STARTER", i, l.Intent)
		}
		if l.Level != 1 {
			t.Errorf("listings[%d] level = %d, want 1", i, l.Level)
		}
		if l.CharacterCount != 1 {
			t.Errorf("listings[%d] character_count = %d, want 1", i, l.CharacterCount)
		}
		if len(l.Tags) == 0 {
			t.Errorf("listings[%d] has no tags", i)
		}
	}
}

func TestBuiltinListings_DeepCopy(t *testing.T) {
	a, err := BuiltinListings()
	if err != nil {
		t.Fatalf("BuiltinListings: %v", err)
	}
	a[0].Title = "mutated"
	a[0].Tags[0] = "mutated"

	b, err := BuiltinListings()
	if err != nil {
		t.Fatalf("BuiltinListings: %v", err)
	}
	if b[0].Title == "mutated" {
		t.Error("mutation leaked through to cached data (Title)")
	}
	if b[0].Tags[0] == "mutated" {
		t.Error("mutation leaked through to cached data (Tags)")
	}
}
