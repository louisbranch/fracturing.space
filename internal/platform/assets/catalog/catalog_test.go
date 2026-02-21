package catalog

import (
	"errors"
	"testing"
)

func TestResolveSelection_DefaultsAssetDeterministically(t *testing.T) {
	manifest := AvatarManifest()
	setA, assetA, err := manifest.ResolveSelection(SelectionInput{
		EntityType: "user",
		EntityID:   "user-1",
	})
	if err != nil {
		t.Fatalf("resolve selection: %v", err)
	}
	setB, assetB, err := manifest.ResolveSelection(SelectionInput{
		EntityType: "user",
		EntityID:   "user-1",
	})
	if err != nil {
		t.Fatalf("resolve selection: %v", err)
	}

	if setA == "" || assetA == "" {
		t.Fatalf("expected set/asset defaults, got %q/%q", setA, assetA)
	}
	if setA != setB || assetA != assetB {
		t.Fatalf("expected deterministic result, got %q/%q and %q/%q", setA, assetA, setB, assetB)
	}
}

func TestResolveSelection_RejectsUnknownSet(t *testing.T) {
	manifest := AvatarManifest()
	_, _, err := manifest.ResolveSelection(SelectionInput{
		EntityType: "user",
		EntityID:   "user-1",
		SetID:      "missing",
	})
	if !errors.Is(err, ErrSetNotFound) {
		t.Fatalf("expected ErrSetNotFound, got %v", err)
	}
}

func TestResolveSelection_RejectsUnknownAssetForSet(t *testing.T) {
	manifest := AvatarManifest()
	_, _, err := manifest.ResolveSelection(SelectionInput{
		EntityType: "user",
		EntityID:   "user-1",
		SetID:      AvatarSetV1,
		AssetID:    "missing",
	})
	if !errors.Is(err, ErrAssetInvalid) {
		t.Fatalf("expected ErrAssetInvalid, got %v", err)
	}
}

func TestResolveSelection_AcceptsCanonicalCampaignCoverID(t *testing.T) {
	manifest := CampaignCoverManifest()
	setID, assetID, err := manifest.ResolveSelection(SelectionInput{
		EntityType: "campaign",
		EntityID:   "camp-1",
		SetID:      CampaignCoverSetV1,
		AssetID:    "mountain_pass",
	})
	if err != nil {
		t.Fatalf("resolve selection: %v", err)
	}
	if setID != CampaignCoverSetV1 {
		t.Fatalf("set id = %q, want %q", setID, CampaignCoverSetV1)
	}
	if assetID != "mountain_pass" {
		t.Fatalf("asset id = %q, want %q", assetID, "mountain_pass")
	}
}
