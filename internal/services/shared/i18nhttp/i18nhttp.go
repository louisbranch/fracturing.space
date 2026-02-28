package i18nhttp

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	_ "github.com/louisbranch/fracturing.space/internal/services/shared/i18nmessages"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	// LangParam is the query parameter used to select a language.
	LangParam = "lang"
	// LangCookieName stores the user's language preference.
	LangCookieName = "fs_lang"
)

// LanguageOption represents a supported language option in UI surfaces.
type LanguageOption struct {
	Tag    string
	Label  string
	Active bool
}

// Supported returns the list of supported language tags.
func Supported() []language.Tag {
	return platformi18n.SupportedTags()
}

// Default returns the default language tag.
func Default() language.Tag {
	return platformi18n.DefaultTag()
}

// Printer returns a message printer for the supplied tag.
func Printer(tag language.Tag) *message.Printer {
	return message.NewPrinter(tag)
}

// ResolveTag determines the best language tag for the request.
// The bool indicates whether the lang query param should be persisted as a cookie.
func ResolveTag(r *http.Request) (language.Tag, bool) {
	if r == nil {
		return Default(), false
	}

	if langValue := strings.TrimSpace(r.URL.Query().Get(LangParam)); langValue != "" {
		if tag, ok := platformi18n.ParseTag(langValue); ok {
			return tag, true
		}
	}

	if cookie, err := r.Cookie(LangCookieName); err == nil {
		if tag, ok := platformi18n.ParseTag(cookie.Value); ok {
			return tag, false
		}
	}

	if accept := strings.TrimSpace(r.Header.Get("Accept-Language")); accept != "" {
		if tags, _, err := language.ParseAcceptLanguage(accept); err == nil {
			return platformi18n.MatchTags(tags), false
		}
	}

	return Default(), false
}

// SetLanguageCookie persists the selected language on the response.
func SetLanguageCookie(w http.ResponseWriter, tag language.Tag) {
	if w == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     LangCookieName,
		Value:    tag.String(),
		Path:     "/",
		MaxAge:   int((365 * 24 * time.Hour).Seconds()),
		SameSite: http.SameSiteLaxMode,
	})
}

// NormalizeTag coerces unknown tags to the default supported language.
func NormalizeTag(value string) language.Tag {
	if tag, ok := platformi18n.ParseTag(value); ok {
		return tag
	}
	return platformi18n.DefaultTag()
}

// BuildLanguageOptions returns supported language options with active selection.
func BuildLanguageOptions(supported []language.Tag, activeLang string, labelForTag func(tag language.Tag) string) []LanguageOption {
	options := make([]LanguageOption, 0, len(supported))
	activeTag := NormalizeTag(activeLang)
	for _, tag := range supported {
		label := tag.String()
		if labelForTag != nil {
			if resolved := strings.TrimSpace(labelForTag(tag)); resolved != "" {
				label = resolved
			}
		}
		options = append(options, LanguageOption{
			Tag:    tag.String(),
			Label:  label,
			Active: tag == activeTag,
		})
	}
	return options
}

// ActiveLanguageLabel returns the label for the active language selection.
func ActiveLanguageLabel(options []LanguageOption) string {
	for _, option := range options {
		if option.Active {
			return option.Label
		}
	}
	if len(options) == 0 {
		return ""
	}
	return options[0].Label
}

// LanguageURL returns the current URL with the language param updated.
func LanguageURL(path string, rawQuery string, tag string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		query = url.Values{}
	}
	query.Set(LangParam, tag)
	return (&url.URL{Path: path, RawQuery: query.Encode()}).String()
}

// LanguageKeyLabel maps a language tag to shared language keys.
func LanguageKeyLabel(tag language.Tag) string {
	switch platformi18n.LocaleForTag(tag) {
	case commonv1.Locale_LOCALE_PT_BR:
		return "nav.lang_pt_br"
	case commonv1.Locale_LOCALE_EN_US:
		return "nav.lang_en"
	default:
		return tag.String()
	}
}
