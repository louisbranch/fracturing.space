package templates

import (
	"net/url"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	"golang.org/x/text/language"
)

// LanguageOption represents a supported language option in the UI.
type LanguageOption struct {
	Tag    string
	Label  string
	Active bool
}

// LanguageOptions returns supported language options with active selection.
func LanguageOptions(page PageContext) []LanguageOption {
	options := make([]LanguageOption, 0, len(webi18n.Supported()))
	activeTag := normalizeTag(page.Lang)
	for _, tag := range webi18n.Supported() {
		options = append(options, LanguageOption{
			Tag:    tag.String(),
			Label:  languageLabel(page.Loc, tag),
			Active: tag == activeTag,
		})
	}
	return options
}

// ActiveLanguageLabel returns the label for the active language selection.
func ActiveLanguageLabel(page PageContext) string {
	for _, option := range LanguageOptions(page) {
		if option.Active {
			return option.Label
		}
	}
	options := LanguageOptions(page)
	if len(options) == 0 {
		return ""
	}
	return options[0].Label
}

// LanguageURL returns the current URL with the language param updated.
func LanguageURL(page PageContext, tag string) string {
	path := page.CurrentPath
	if path == "" {
		path = "/"
	}
	query, err := url.ParseQuery(page.CurrentQuery)
	if err != nil {
		query = url.Values{}
	}
	query.Set(webi18n.LangParam, tag)
	return (&url.URL{Path: path, RawQuery: query.Encode()}).String()
}

// languageLabel maps a language tag to a localized display label.
func languageLabel(loc Localizer, tag language.Tag) string {
	switch platformi18n.LocaleForTag(tag) {
	case commonv1.Locale_LOCALE_PT_BR:
		return T(loc, "nav.lang_pt_br")
	case commonv1.Locale_LOCALE_EN_US:
		return T(loc, "nav.lang_en")
	default:
		return tag.String()
	}
}

// normalizeTag coerces unknown tags to the default supported language.
func normalizeTag(value string) language.Tag {
	if tag, ok := platformi18n.ParseTag(value); ok {
		return tag
	}
	return platformi18n.DefaultTag()
}
