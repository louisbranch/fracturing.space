package account

import (
	"errors"
	"testing"
	"time"
)

func TestNewProfile_DefaultsAvatarSelectionDeterministically(t *testing.T) {
	now := time.Date(2026, 2, 19, 0, 0, 0, 0, time.UTC)
	profileA, err := NewProfile(ProfileInput{
		UserID: "user-1",
		Name:   "Alice",
	}, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new profile: %v", err)
	}
	profileB, err := NewProfile(ProfileInput{
		UserID: " user-1 ",
		Name:   "Alice",
	}, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new profile: %v", err)
	}

	if profileA.AvatarSetID == "" {
		t.Fatal("expected avatar set id to be defaulted")
	}
	if profileA.AvatarAssetID == "" {
		t.Fatal("expected avatar asset id to be defaulted")
	}
	if profileA.AvatarSetID != profileB.AvatarSetID {
		t.Fatalf("avatar set mismatch: %q vs %q", profileA.AvatarSetID, profileB.AvatarSetID)
	}
	if profileA.AvatarAssetID != profileB.AvatarAssetID {
		t.Fatalf("avatar asset mismatch: %q vs %q", profileA.AvatarAssetID, profileB.AvatarAssetID)
	}
}

func TestNewProfile_RejectsInvalidAvatarSet(t *testing.T) {
	_, err := NewProfile(ProfileInput{
		UserID:      "user-1",
		AvatarSetID: "missing",
	}, time.Now)
	if !errors.Is(err, ErrAvatarSetInvalid) {
		t.Fatalf("expected ErrAvatarSetInvalid, got %v", err)
	}
}

func TestNewProfile_RejectsInvalidAvatarAsset(t *testing.T) {
	_, err := NewProfile(ProfileInput{
		UserID:        "user-1",
		AvatarAssetID: "missing",
	}, time.Now)
	if !errors.Is(err, ErrAvatarAssetInvalid) {
		t.Fatalf("expected ErrAvatarAssetInvalid, got %v", err)
	}
}
