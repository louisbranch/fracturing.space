package app

import (
	"strings"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
)

// campaignLocaleCanonical normalizes locale labels/tags used by web forms and views.
func campaignLocaleCanonical(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	switch strings.ToLower(trimmed) {
	case "english (us)":
		return "en-US"
	case "portuguese (brazil)":
		return "pt-BR"
	}
	locale, ok := platformi18n.ParseLocale(trimmed)
	if !ok {
		return ""
	}
	return platformi18n.LocaleString(locale)
}
