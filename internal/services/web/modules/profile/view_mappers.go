package profile

import webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"

// mapPublicProfileTemplateView maps values across transport and template boundaries.
func mapPublicProfileTemplateView(profile Profile, viewerSignedIn bool) webtemplates.PublicProfileView {
	return webtemplates.PublicProfileView{
		Username:       profile.Username,
		Name:           profile.Name,
		Pronouns:       profile.Pronouns,
		Bio:            profile.Bio,
		AvatarURL:      profile.AvatarURL,
		ViewerSignedIn: viewerSignedIn,
	}
}
