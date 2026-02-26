// Package i18n provides request language helpers for web2 handlers.
package i18n

import (
	"net/http"
	"strings"
)

const defaultLanguage = "en"

// ResolveLanguage returns a two-letter language code for the request.
func ResolveLanguage(r *http.Request) string {
	if r == nil {
		return defaultLanguage
	}
	header := strings.TrimSpace(r.Header.Get("Accept-Language"))
	if header == "" {
		return defaultLanguage
	}
	first := strings.Split(header, ",")[0]
	first = strings.TrimSpace(strings.Split(first, ";")[0])
	if first == "" {
		return defaultLanguage
	}
	first = strings.ToLower(first)
	if len(first) >= 2 {
		return first[:2]
	}
	return defaultLanguage
}
