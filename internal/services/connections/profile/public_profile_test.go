package profile

import (
	"strings"
	"testing"
)

func TestNormalize_ValidInput(t *testing.T) {
	got, err := Normalize("  Alice  ", "https://cdn.example.com/avatar.png", "  Campaign manager  ")
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if got.DisplayName != "Alice" {
		t.Fatalf("display_name = %q, want Alice", got.DisplayName)
	}
	if got.AvatarURL != "https://cdn.example.com/avatar.png" {
		t.Fatalf("avatar_url = %q, want https://cdn.example.com/avatar.png", got.AvatarURL)
	}
	if got.Bio != "Campaign manager" {
		t.Fatalf("bio = %q, want Campaign manager", got.Bio)
	}
}

func TestNormalize_DisplayNameTooLongReturnsError(t *testing.T) {
	_, err := Normalize(strings.Repeat("a", maxDisplayNameLength+1), "", "")
	if err == nil {
		t.Fatal("expected display_name length validation error")
	}
}

func TestNormalize_AvatarURLRequiresHTTPOrHTTPS(t *testing.T) {
	_, err := Normalize("Alice", "ftp://cdn.example.com/avatar.png", "")
	if err == nil {
		t.Fatal("expected avatar_url scheme validation error")
	}
}

func TestNormalize_AvatarURLTooLongReturnsError(t *testing.T) {
	avatarURL := "https://cdn.example.com/" + strings.Repeat("a", maxAvatarURLLength)
	_, err := Normalize("Alice", avatarURL, "")
	if err == nil {
		t.Fatal("expected avatar_url length validation error")
	}
}

func TestNormalize_BioTooLongReturnsError(t *testing.T) {
	_, err := Normalize("Alice", "", strings.Repeat("a", maxBioLength+1))
	if err == nil {
		t.Fatal("expected bio length validation error")
	}
}
