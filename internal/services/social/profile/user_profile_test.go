package profile

import (
	"strings"
	"testing"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
)

func TestNormalize_ValidInput(t *testing.T) {
	got, err := Normalize("user-1", "  Alice  ", "avatar_set_v1", "001", "  Campaign manager  ", " she / her ")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if got.Name != "Alice" {
		t.Fatalf("name = %q, want Alice", got.Name)
	}
	if got.AvatarSetID != "avatar_set_v1" {
		t.Fatalf("avatar_set_id = %q, want avatar_set_v1", got.AvatarSetID)
	}
	if got.AvatarAssetID != "001" {
		t.Fatalf("avatar_asset_id = %q, want 001", got.AvatarAssetID)
	}
	if got.Bio != "Campaign manager" {
		t.Fatalf("bio = %q, want Campaign manager", got.Bio)
	}
	if got.Pronouns != "she / her" {
		t.Fatalf("pronouns = %q, want she / her", got.Pronouns)
	}
}

func TestNormalize_DefaultAvatarUsesPeopleSet(t *testing.T) {
	expectedSetID, expectedAssetID, err := assetcatalog.AvatarManifest().ResolveSelection(assetcatalog.SelectionInput{
		EntityType: assetcatalog.AvatarRoleUser,
		EntityID:   "user-1",
		SetID:      assetcatalog.AvatarSetPeopleV1,
	})
	if err != nil {
		t.Fatalf("resolve expected avatar: %v", err)
	}

	got, err := Normalize("user-1", "", "", "", "", "")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if got.Name != "" {
		t.Fatalf("name = %q, want empty", got.Name)
	}
	if got.AvatarSetID != expectedSetID {
		t.Fatalf("avatar_set_id = %q, want %q", got.AvatarSetID, expectedSetID)
	}
	if got.AvatarAssetID != expectedAssetID {
		t.Fatalf("avatar_asset_id = %q, want %q", got.AvatarAssetID, expectedAssetID)
	}
}

func TestNormalize_NameTooLongReturnsError(t *testing.T) {
	_, err := Normalize("user-1", strings.Repeat("a", maxNameLength+1), "", "", "", "")
	if err == nil {
		t.Fatal("expected name length validation error")
	}
}

func TestNormalize_AvatarAssetWithoutSetReturnsError(t *testing.T) {
	_, err := Normalize("user-1", "Alice", "", "001", "", "")
	if err == nil {
		t.Fatal("expected avatar validation error")
	}
}

func TestNormalize_AvatarSetWithoutAssetUsesDeterministicSelection(t *testing.T) {
	got, err := Normalize("user-1", "Alice", assetcatalog.AvatarSetPeopleV1, "", "", "")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if got.AvatarSetID != assetcatalog.AvatarSetPeopleV1 {
		t.Fatalf("avatar_set_id = %q, want %q", got.AvatarSetID, assetcatalog.AvatarSetPeopleV1)
	}
	if got.AvatarAssetID == "" {
		t.Fatal("avatar_asset_id = empty, want deterministic people-set asset")
	}
}

func TestNormalize_InvalidAvatarSetReturnsError(t *testing.T) {
	_, err := Normalize("user-1", "Alice", "missing", "001", "", "")
	if err == nil {
		t.Fatal("expected avatar set validation error")
	}
}

func TestNormalize_BioTooLongReturnsError(t *testing.T) {
	_, err := Normalize("user-1", "Alice", "", "", strings.Repeat("a", maxBioLength+1), "")
	if err == nil {
		t.Fatal("expected bio length validation error")
	}
}
