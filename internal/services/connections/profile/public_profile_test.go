package profile

import (
	"strings"
	"testing"
)

func TestNormalize_ValidInput(t *testing.T) {
	got, err := Normalize("user-1", "  Alice  ", "avatar_set_v1", "001", "  Campaign manager  ")
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
}

func TestNormalize_NameTooLongReturnsError(t *testing.T) {
	_, err := Normalize("user-1", strings.Repeat("a", maxNameLength+1), "", "", "")
	if err == nil {
		t.Fatal("expected name length validation error")
	}
}

func TestNormalize_AvatarRequiresPair(t *testing.T) {
	_, err := Normalize("user-1", "Alice", "avatar_set_v1", "", "")
	if err == nil {
		t.Fatal("expected avatar pair validation error")
	}
}

func TestNormalize_InvalidAvatarSetReturnsError(t *testing.T) {
	_, err := Normalize("user-1", "Alice", "missing", "001", "")
	if err == nil {
		t.Fatal("expected avatar set validation error")
	}
}

func TestNormalize_BioTooLongReturnsError(t *testing.T) {
	_, err := Normalize("user-1", "Alice", "", "", strings.Repeat("a", maxBioLength+1))
	if err == nil {
		t.Fatal("expected bio length validation error")
	}
}
