package listing

import (
	"testing"

	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
)

func TestProtoContract_CampaignListingServiceSymbolsExist(t *testing.T) {
	var _ listingv1.CampaignListingServiceServer
	if listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_UNSPECIFIED != 0 {
		t.Fatal("unexpected enum baseline")
	}
}

func TestProtoContract_CampaignListingGmModeEnumValues(t *testing.T) {
	if listingv1.CampaignListingGmMode_CAMPAIGN_LISTING_GM_MODE_UNSPECIFIED != 0 {
		t.Fatal("gm mode unspecified should be 0")
	}
	if listingv1.CampaignListingGmMode_CAMPAIGN_LISTING_GM_MODE_AI != 1 {
		t.Fatal("gm mode AI should be 1")
	}
	if listingv1.CampaignListingGmMode_CAMPAIGN_LISTING_GM_MODE_HUMAN != 2 {
		t.Fatal("gm mode HUMAN should be 2")
	}
	if listingv1.CampaignListingGmMode_CAMPAIGN_LISTING_GM_MODE_HYBRID != 3 {
		t.Fatal("gm mode HYBRID should be 3")
	}
}

func TestProtoContract_CampaignListingIntentEnumValues(t *testing.T) {
	if listingv1.CampaignListingIntent_CAMPAIGN_LISTING_INTENT_UNSPECIFIED != 0 {
		t.Fatal("intent unspecified should be 0")
	}
	if listingv1.CampaignListingIntent_CAMPAIGN_LISTING_INTENT_STARTER != 2 {
		t.Fatal("intent starter should be 2")
	}
}

func TestProtoContract_CampaignListingNewFieldNumbers(t *testing.T) {
	listing := &listingv1.CampaignListing{
		GmMode:         listingv1.CampaignListingGmMode_CAMPAIGN_LISTING_GM_MODE_AI,
		Intent:         listingv1.CampaignListingIntent_CAMPAIGN_LISTING_INTENT_STARTER,
		Level:          1,
		CharacterCount: 1,
		Storyline:      "test storyline",
		Tags:           []string{"solo", "beginner"},
	}
	if listing.GetGmMode() != listingv1.CampaignListingGmMode_CAMPAIGN_LISTING_GM_MODE_AI {
		t.Fatal("gm_mode field 11 round-trip failed")
	}
	if listing.GetIntent() != listingv1.CampaignListingIntent_CAMPAIGN_LISTING_INTENT_STARTER {
		t.Fatal("intent field 12 round-trip failed")
	}
	if listing.GetLevel() != 1 {
		t.Fatal("level field 13 round-trip failed")
	}
	if listing.GetCharacterCount() != 1 {
		t.Fatal("character_count field 14 round-trip failed")
	}
	if listing.GetStoryline() != "test storyline" {
		t.Fatal("storyline field 15 round-trip failed")
	}
	if len(listing.GetTags()) != 2 || listing.GetTags()[0] != "solo" {
		t.Fatal("tags field 16 round-trip failed")
	}
}
