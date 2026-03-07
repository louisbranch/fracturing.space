package websupport

import (
	"regexp"
	"testing"
)

func TestAvatarImageURL_UsesCloudinaryPublicIDPath(t *testing.T) {
	got := AvatarImageURL(
		"https://res.cloudinary.com/fracturing-space/image/upload",
		"user",
		"user-1",
		"avatar_set_v1",
		"apothecary_journeyman",
	)
	pattern := regexp.MustCompile(`^https://res\.cloudinary\.com/fracturing-space/image/upload/c_crop,w_512,h_768,x_0,y_0/f_auto,q_auto,dpr_auto,c_limit,w_192/v[0-9]+/high_fantasy/avatar_set/v1/apothecary_journeyman\.png$`)
	if !pattern.MatchString(got) {
		t.Fatalf("AvatarImageURL(...) = %q, want match %q", got, pattern.String())
	}
}
