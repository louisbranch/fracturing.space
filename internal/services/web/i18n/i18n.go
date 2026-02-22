// Package i18n provides locale resolution and message printing for the web login service.
package i18n

import (
	"net/http"

	sharedi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	// LangParam is the query parameter used to select a language.
	LangParam = sharedi18n.LangParam
	// LangCookieName stores the user's language preference.
	LangCookieName = sharedi18n.LangCookieName
)

// Supported returns the list of supported language tags.
func Supported() []language.Tag {
	return sharedi18n.Supported()
}

// Default returns the default language tag.
func Default() language.Tag {
	return sharedi18n.Default()
}

// Printer returns a message printer for the supplied tag.
func Printer(tag language.Tag) *message.Printer {
	return sharedi18n.Printer(tag)
}

// ResolveTag determines the best language tag for the request.
// The bool indicates whether the lang query param should be persisted as a cookie.
func ResolveTag(r *http.Request) (language.Tag, bool) {
	return sharedi18n.ResolveTag(r)
}

// SetLanguageCookie persists the selected language on the response.
func SetLanguageCookie(w http.ResponseWriter, tag language.Tag) {
	sharedi18n.SetLanguageCookie(w, tag)
}
