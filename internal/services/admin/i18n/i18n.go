package i18n

import (
	"net/http"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	// LangParam is the query parameter used to select a language.
	LangParam = "lang"
	// LangCookieName stores the user's language preference.
	LangCookieName = "fs_lang"
)

var supportedTags = []language.Tag{
	language.English,
	language.MustParse("pt-BR"),
}

var tagMatcher = language.NewMatcher(supportedTags)
var supportedTagSet = make(map[string]language.Tag, len(supportedTags))

func init() {
	for _, tag := range supportedTags {
		supportedTagSet[tag.String()] = tag
	}
}

// Supported returns the list of supported language tags.
func Supported() []language.Tag {
	tags := make([]language.Tag, len(supportedTags))
	copy(tags, supportedTags)
	return tags
}

// Default returns the default language tag.
func Default() language.Tag {
	return language.English
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
		if tag, ok := parseTag(langValue); ok {
			return tag, true
		}
	}

	if cookie, err := r.Cookie(LangCookieName); err == nil {
		if tag, ok := parseTag(cookie.Value); ok {
			return tag, false
		}
	}

	if accept := strings.TrimSpace(r.Header.Get("Accept-Language")); accept != "" {
		if tags, _, err := language.ParseAcceptLanguage(accept); err == nil {
			matched, _, _ := tagMatcher.Match(tags...)
			return matched, false
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

func parseTag(value string) (language.Tag, bool) {
	parsed, err := language.Parse(value)
	if err != nil {
		return language.Tag{}, false
	}
	if tag, ok := supportedTagSet[parsed.String()]; ok {
		return tag, true
	}
	return language.Tag{}, false
}
