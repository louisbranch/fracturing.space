package catalog

import (
	"regexp"
	"testing"
)

func TestResolveCDNAssetID_UsesVersionedCloudinaryCampaignPath(t *testing.T) {
	got := ResolveCDNAssetID(CampaignCoverSetV1, "ashen_city_gate")
	pattern := regexp.MustCompile(`^v[0-9]+/high_fantasy/campaign_scene/v1/ashen_city_gate$`)
	if !pattern.MatchString(got) {
		t.Fatalf("ResolveCDNAssetID(...) = %q, want match %q", got, pattern.String())
	}
}

func TestResolveCDNAssetID_UsesVersionedCloudinaryAvatarPath(t *testing.T) {
	got := ResolveCDNAssetID(AvatarSetPeopleV1, "apothecary_journeyman")
	pattern := regexp.MustCompile(`^v[0-9]+/high_fantasy/avatar_set/v1/apothecary_journeyman$`)
	if !pattern.MatchString(got) {
		t.Fatalf("ResolveCDNAssetID(...) = %q, want match %q", got, pattern.String())
	}
}

func TestResolveCDNAssetID_FallsBackToCanonicalAssetID(t *testing.T) {
	got := ResolveCDNAssetID("unknown_set", "unknown_asset")
	want := "unknown_asset"
	if got != want {
		t.Fatalf("ResolveCDNAssetID(...) = %q, want %q", got, want)
	}
}

func TestCloudinaryPublicID_RejectsMissingSelectors(t *testing.T) {
	if _, ok := CloudinaryPublicID("", "ashen_city_gate"); ok {
		t.Fatal("expected missing set id to fail")
	}
	if _, ok := CloudinaryPublicID(CampaignCoverSetV1, ""); ok {
		t.Fatal("expected missing asset id to fail")
	}
}
