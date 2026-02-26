package web

import (
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	websupport "github.com/louisbranch/fracturing.space/internal/services/web/support"
)

func TestAvatarImageURL_CloudinaryParticipantUsesFirstPortraitCrop(t *testing.T) {
	got := websupport.AvatarImageURL(
		"https://res.cloudinary.com/fracturing-space/image/upload",
		"participant",
		"part-1",
		"avatar_set_v1",
		"001",
	)
	want := "https://res.cloudinary.com/fracturing-space/image/upload/c_crop,w_512,h_768,x_0,y_0/f_auto,q_auto,dpr_auto,c_limit,w_192/001.png"
	if got != want {
		t.Fatalf("avatarImageURL(...) = %q, want %q", got, want)
	}
}

func TestAvatarImageURL_CloudinaryCharacterUsesDeterministicVariantPortrait(t *testing.T) {
	gotA := websupport.AvatarImageURL(
		"https://res.cloudinary.com/fracturing-space/image/upload",
		"character",
		"char-1",
		"avatar_set_v1",
		"007",
	)
	gotB := websupport.AvatarImageURL(
		"https://res.cloudinary.com/fracturing-space/image/upload",
		"character",
		"char-1",
		"avatar_set_v1",
		"007",
	)
	if gotA != gotB {
		t.Fatalf("avatarImageURL deterministic mismatch: %q vs %q", gotA, gotB)
	}
	if !strings.HasSuffix(gotA, "/007.png") {
		t.Fatalf("avatarImageURL suffix = %q, want /007.png", gotA)
	}

	valid := []string{
		"/c_crop,w_512,h_768,x_512,y_0/",
		"/c_crop,w_512,h_768,x_0,y_768/",
		"/c_crop,w_512,h_768,x_512,y_768/",
	}
	for _, candidate := range valid {
		if strings.Contains(gotA, candidate) {
			return
		}
	}
	t.Fatalf("avatarImageURL crop segment = %q, want one of %v", gotA, valid)
}

func TestAvatarImageURL_NonCloudinaryBaseUsesFlatAssetPath(t *testing.T) {
	got := websupport.AvatarImageURL(
		"https://cdn.example.com/avatars",
		"user",
		"user-1",
		"avatar_set_v1",
		"001",
	)
	want := "https://cdn.example.com/avatars/001.png"
	if got != want {
		t.Fatalf("avatarImageURL(...) = %q, want %q", got, want)
	}
}

func TestAvatarImageURL_RespectsExplicitBlankAvatarSelection(t *testing.T) {
	got := websupport.AvatarImageURL(
		"https://cdn.example.com/avatars",
		"participant",
		"part-1",
		catalog.AvatarSetBlankV1,
		"000",
	)
	want := "https://cdn.example.com/avatars/000.png"
	if got != want {
		t.Fatalf("avatarImageURL(...) = %q, want %q", got, want)
	}
}

func TestAvatarImageURL_LegacyPeopleSetBlankSelectionFallsBackToBlankSet(t *testing.T) {
	got := websupport.AvatarImageURL(
		"https://cdn.example.com/avatars",
		"participant",
		"part-legacy",
		catalog.AvatarSetPeopleV1,
		catalog.AvatarAssetBlank,
	)
	want := "https://cdn.example.com/avatars/000.png"
	if got != want {
		t.Fatalf("avatarImageURL(...) = %q, want %q", got, want)
	}
}
