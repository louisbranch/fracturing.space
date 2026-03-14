package app

import (
	"regexp"
	"testing"
)

func TestCampaignCoverImageURL_UsesCloudinaryPublicIDPath(t *testing.T) {
	got := CampaignCoverImageURL(
		"https://res.cloudinary.com/fracturing-space/image/upload",
		"campaign-1",
		"campaign_cover_set_v1",
		"ashen_city_gate",
	)
	pattern := regexp.MustCompile(`^https://res\.cloudinary\.com/fracturing-space/image/upload/f_auto,q_auto,dpr_auto,c_limit,w_640/v[0-9]+/high_fantasy/campaign_scene/v1/ashen_city_gate\.png$`)
	if !pattern.MatchString(got) {
		t.Fatalf("CampaignCoverImageURL(...) = %q, want match %q", got, pattern.String())
	}
}

func TestCampaignCoverPreviewImageURL_UsesSmallerDeliveryWidth(t *testing.T) {
	got := CampaignCoverPreviewImageURL(
		"https://res.cloudinary.com/fracturing-space/image/upload",
		"campaign-1",
		"campaign_cover_set_v1",
		"ashen_city_gate",
	)
	pattern := regexp.MustCompile(`^https://res\.cloudinary\.com/fracturing-space/image/upload/f_auto,q_auto,dpr_auto,c_limit,w_128/v[0-9]+/high_fantasy/campaign_scene/v1/ashen_city_gate\.png$`)
	if !pattern.MatchString(got) {
		t.Fatalf("CampaignCoverPreviewImageURL(...) = %q, want match %q", got, pattern.String())
	}
}
