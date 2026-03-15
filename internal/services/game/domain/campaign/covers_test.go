package campaign

import "testing"

func TestNormalizeCampaignCoverAssetID_RejectsBlankAndLegacyValues(t *testing.T) {
	if _, ok := normalizeCampaignCoverAssetID("   "); ok {
		t.Fatal("expected blank cover ids to be rejected")
	}
	if _, ok := normalizeCampaignCoverAssetID("camp-cover-01"); ok {
		t.Fatal("expected legacy SVG cover ids to be rejected")
	}
}

func TestNormalizeCampaignCoverAssetID_AcceptsCanonicalValues(t *testing.T) {
	got, ok := normalizeCampaignCoverAssetID("ashen_city_gate")
	if !ok {
		t.Fatal("expected ashen_city_gate to normalize successfully")
	}
	if got != "ashen_city_gate" {
		t.Fatalf("normalizeCampaignCoverAssetID(ashen_city_gate) = %q, want %q", got, "ashen_city_gate")
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
	if _, ok := normalizeCampaignCoverAssetID(first); !ok {
		t.Fatalf("expected deterministic default %q to be in catalog", first)
	}
}

func TestDefaultCampaignCoverAssetID_EmptyCatalogReturnsBlank(t *testing.T) {
	originalCatalog := campaignCoverAssetCatalog
	campaignCoverAssetCatalog = nil
	t.Cleanup(func() {
		campaignCoverAssetCatalog = originalCatalog
	})

	if got := defaultCampaignCoverAssetID("camp-1"); got != "" {
		t.Fatalf("defaultCampaignCoverAssetID() = %q, want blank when catalog is empty", got)
	}
}
