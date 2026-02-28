package templates

import "strings"

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
