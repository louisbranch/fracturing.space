package profile

import (
	"strings"

	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// participantPronounsLabel preserves the shared public-profile pronoun display
// mapping.
func participantPronounsLabel(loc webtemplates.Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return raw
	case "she/her":
		return webtemplates.T(loc, "game.participants.value_she_her")
	case "he/him":
		return webtemplates.T(loc, "game.participants.value_he_him")
	case "they/them":
		return webtemplates.T(loc, "game.participants.value_they_them")
	case "it/its":
		return webtemplates.T(loc, "game.participants.value_it_its")
	default:
		return raw
	}
}
