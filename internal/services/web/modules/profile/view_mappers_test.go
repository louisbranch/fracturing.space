package profile

import (
	"testing"

	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
)

func TestMapPublicProfileTemplateViewBuildsAvatarURLFromProfileIdentity(t *testing.T) {
	t.Parallel()

	view := mapPublicProfileTemplateView(profileapp.Profile{
		Username:      "louis",
		UserID:        "user-1",
		Name:          "Louis Branch",
		Pronouns:      "they/them",
		Bio:           "Explorer",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "apothecary_journeyman",
	}, "https://cdn.example.com/avatars", true)

	wantAvatarURL := websupport.AvatarImageURL(
		"https://cdn.example.com/avatars",
		"user",
		"user-1",
		"avatar_set_v1",
		"apothecary_journeyman",
	)
	if view.AvatarURL != wantAvatarURL {
		t.Fatalf("AvatarURL = %q, want %q", view.AvatarURL, wantAvatarURL)
	}
	if !view.ViewerSignedIn {
		t.Fatal("ViewerSignedIn = false, want true")
	}
}

func TestMapPublicProfileTemplateViewFallsBackToUsernameWhenUserIDMissing(t *testing.T) {
	t.Parallel()

	view := mapPublicProfileTemplateView(profileapp.Profile{
		Username:      " louis ",
		AvatarSetID:   "avatar_set_v1",
		AvatarAssetID: "apothecary_journeyman",
	}, "https://cdn.example.com/avatars", false)

	wantAvatarURL := websupport.AvatarImageURL(
		"https://cdn.example.com/avatars",
		"user",
		"louis",
		"avatar_set_v1",
		"apothecary_journeyman",
	)
	if view.AvatarURL != wantAvatarURL {
		t.Fatalf("AvatarURL = %q, want %q", view.AvatarURL, wantAvatarURL)
	}
	if view.ViewerSignedIn {
		t.Fatal("ViewerSignedIn = true, want false")
	}
}
