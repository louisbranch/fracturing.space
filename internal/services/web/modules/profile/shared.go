package profile

import (
	"strings"

	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/message"
)

// Localizer keeps profile-owned templates aligned with the shared translation seam.
type Localizer = webtemplates.Localizer

// AppImageView reuses the shared responsive image contract.
type AppImageView = webtemplates.AppImageView

// T keeps profile-owned template translation lookups on the shared helper.
func T(loc Localizer, key message.Reference, args ...any) string {
	return webtemplates.T(loc, key, args...)
}

// AppImage reuses the shared image component without moving image ownership.
func AppImage(view AppImageView) templ.Component {
	return webtemplates.AppImage(view)
}

// participantPronounsLabel preserves the shared public-profile pronoun display mapping.
func participantPronounsLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return raw
	case "she/her":
		return T(loc, "game.participants.value_she_her")
	case "he/him":
		return T(loc, "game.participants.value_he_him")
	case "they/them":
		return T(loc, "game.participants.value_they_them")
	case "it/its":
		return T(loc, "game.participants.value_it_its")
	default:
		return raw
	}
}
