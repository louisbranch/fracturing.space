// Package i18n provides locale resolution and message printing for the web login service.
package i18n

import (
	"net/http"
	"strings"
	"time"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	// LangParam is the query parameter used to select a language.
	LangParam = "lang"
	// LangCookieName stores the user's language preference.
	LangCookieName = "fs_lang"
)

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
