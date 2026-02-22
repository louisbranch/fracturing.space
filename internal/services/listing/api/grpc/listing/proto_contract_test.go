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
