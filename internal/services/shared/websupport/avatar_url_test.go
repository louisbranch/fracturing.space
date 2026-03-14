package websupport

import (
	"regexp"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
)

func TestAvatarImageURL_UsesCloudinaryPublicIDPath(t *testing.T) {
	got := AvatarImageURL(
		"https://res.cloudinary.com/fracturing-space/image/upload",
		"user",
		"user-1",
		"avatar_set_v1",
		"apothecary_journeyman",
		192,
	)
	pattern := regexp.MustCompile(`^https://res\.cloudinary\.com/fracturing-space/image/upload/c_crop,w_512,h_768,x_0,y_0/f_auto,q_auto,dpr_auto,c_limit,w_192/v[0-9]+/high_fantasy/avatar_set/v1/apothecary_journeyman\.png$`)
	if !pattern.MatchString(got) {
		t.Fatalf("AvatarImageURL(...) = %q, want match %q", got, pattern.String())
	}
}

func TestResolveWebAvatarSelection_DefaultsUsersToPeopleSet(t *testing.T) {
	setID, assetID := ResolveWebAvatarSelection("user", "user-1", "", "")
	if setID != catalog.AvatarSetPeopleV1 {
		t.Fatalf("set_id = %q, want %q", setID, catalog.AvatarSetPeopleV1)
	}
	if !catalog.AvatarManifest().ValidateAssetInSet(setID, assetID) {
		t.Fatalf("asset_id = %q, want asset in %q", assetID, setID)
	}
}

func TestResolveWebAvatarSelection_KeepsBlankDefaultForParticipants(t *testing.T) {
	setID, assetID := ResolveWebAvatarSelection("participant", "participant-1", "", "")
	if setID != catalog.AvatarSetBlankV1 {
		t.Fatalf("set_id = %q, want %q", setID, catalog.AvatarSetBlankV1)
	}
	if !catalog.AvatarManifest().ValidateAssetInSet(setID, assetID) {
		t.Fatalf("asset_id = %q, want asset in %q", assetID, setID)
	}
}
