package profile

import (
	"strings"

	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
)

const publicProfileAvatarDeliveryWidthPX = 512

// mapPublicProfileTemplateView maps values across transport and template boundaries.
func mapPublicProfileTemplateView(profile profileapp.Profile, assetBaseURL string, viewerSignedIn bool) PublicProfileView {
	entityID := strings.TrimSpace(profile.UserID)
	if entityID == "" {
		entityID = strings.TrimSpace(profile.Username)
	}

	return PublicProfileView{
		Username: profile.Username,
		Name:     profile.Name,
		Pronouns: profile.Pronouns,
		Bio:      profile.Bio,
		AvatarURL: websupport.AvatarImageURL(
			strings.TrimSpace(assetBaseURL),
			"user",
			entityID,
			strings.TrimSpace(profile.AvatarSetID),
			strings.TrimSpace(profile.AvatarAssetID),
			publicProfileAvatarDeliveryWidthPX,
		),
		ViewerSignedIn: viewerSignedIn,
	}
}
