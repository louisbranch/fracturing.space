package templates

import (
	sharedi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	"golang.org/x/text/language"
)

// LanguageOption represents a supported language option in the UI.
type LanguageOption = sharedi18n.LanguageOption

// LanguageOptions returns supported language options with active selection.
func LanguageOptions(page PageContext) []LanguageOption {
	return sharedi18n.BuildLanguageOptions(webi18n.Supported(), page.Lang, func(tag language.Tag) string {
		return T(page.Loc, sharedi18n.LanguageKeyLabel(tag))
	})
}

// ActiveLanguageLabel returns the label for the active language selection.
func ActiveLanguageLabel(page PageContext) string {
	return sharedi18n.ActiveLanguageLabel(LanguageOptions(page))
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
