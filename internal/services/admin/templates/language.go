package templates

import (
	admini18n "github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	sharedi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"golang.org/x/text/language"
)

// LanguageOption represents a supported language option in the admin UI.
type LanguageOption = sharedi18n.LanguageOption

// LanguageOptions returns supported language options with active selection.
func LanguageOptions(page PageContext, loc Localizer) []LanguageOption {
	return sharedi18n.BuildLanguageOptions(admini18n.Supported(), page.Lang, func(tag language.Tag) string {
		return T(loc, sharedi18n.LanguageKeyLabel(tag))
	})
}

// ActiveLanguageLabel returns the label for the active language selection.
func ActiveLanguageLabel(page PageContext, loc Localizer) string {
	return sharedi18n.ActiveLanguageLabel(LanguageOptions(page, loc))
}

// LanguageURL returns the current URL with the language param updated.
func LanguageURL(page PageContext, tag string) string {
	return sharedi18n.LanguageURL(page.CurrentPath, page.CurrentQuery, tag)
}

// languageLabel maps a language tag to a localized display label.
func languageLabel(loc Localizer, tag language.Tag) string {
	return T(loc, sharedi18n.LanguageKeyLabel(tag))
}

// normalizeTag coerces unknown tags to the default supported language.
func normalizeTag(value string) language.Tag {
	return sharedi18n.NormalizeTag(value)
}
