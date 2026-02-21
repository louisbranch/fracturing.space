package campaign

import "testing"

func TestIsCampaignCoverAssetID_AcceptsAllConfiguredPNGAssets(t *testing.T) {
	for _, coverAssetID := range campaignCoverAssetCatalog {
		if !isCampaignCoverAssetID(coverAssetID) {
			t.Fatalf("expected %q to be a valid campaign cover asset id", coverAssetID)
		}
	}
}

func TestIsCampaignCoverAssetID_RejectsLegacySVGIDs(t *testing.T) {
	if isCampaignCoverAssetID("camp-cover-01") {
		t.Fatal("expected camp-cover-01 to be rejected after migrating to PNG cover ids")
	}
}

func TestIsCampaignCoverAssetID_RejectsBlankValues(t *testing.T) {
	if isCampaignCoverAssetID("   ") {
		t.Fatal("expected blank cover ids to be rejected")
	}
}

func TestDefaultCampaignCoverAssetID_EmptyCampaignIDReturnsFirstConfiguredAsset(t *testing.T) {
	got := defaultCampaignCoverAssetID("   ")
	if got != campaignCoverAssetCatalog[0] {
		t.Fatalf("defaultCampaignCoverAssetID(blank) = %q, want %q", got, campaignCoverAssetCatalog[0])
	}
}

func TestDefaultCampaignCoverAssetID_IsDeterministicForCampaignID(t *testing.T) {
	first := defaultCampaignCoverAssetID("camp-1")
	second := defaultCampaignCoverAssetID(" camp-1 ")
	if first == "" {
		t.Fatal("expected deterministic default cover id to be non-empty")
	}
	if first != second {
		t.Fatalf("defaultCampaignCoverAssetID mismatch: %q vs %q", first, second)
	}
	if !isCampaignCoverAssetID(first) {
		t.Fatalf("expected deterministic default %q to be in catalog", first)
	}
}
