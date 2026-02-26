package support

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	sharedhtmx "github.com/louisbranch/fracturing.space/internal/services/shared/htmx"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

const gamePageContentType = "text/html; charset=utf-8"

// WriteGameContentType sets the shared HTML content type for web rendering responses.
func WriteGameContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", gamePageContentType)
}

// ErrNoWebPageComponent is returned when a page renderer receives a nil component.
var ErrNoWebPageComponent = errors.New("web: no page component provided")

// WritePage renders a templ component with optional HTMX title injection.
func WritePage(w http.ResponseWriter, r *http.Request, page templ.Component, htmxTitle string) error {
	WriteGameContentType(w)
	if page == nil {
		return ErrNoWebPageComponent
	}
	if sharedhtmx.IsHTMXRequest(r) {
		sharedhtmx.RenderPage(w, r, page, page, htmxTitle)
		return nil
	}
	return page.Render(r.Context(), w)
}

// ComposeHTMXTitle builds a title tag for page fragments and full-page renders.
func ComposeHTMXTitle(loc webtemplates.Localizer, title string, args ...any) string {
	if loc == nil {
		return sharedhtmx.TitleTag(sharedtemplates.ComposePageTitle(title))
	}
	return sharedhtmx.TitleTag(sharedtemplates.ComposePageTitle(webtemplates.T(loc, title, args...)))
}

// ComposeHTMXTitleForPage builds an HTMX title from a page context and title key.
func ComposeHTMXTitleForPage(page webtemplates.PageContext, title string, args ...any) string {
	return ComposeHTMXTitle(page.Loc, title, args...)
}

// ResolveChatFallbackPort extracts the final component of a host:port pair.
func ResolveChatFallbackPort(rawAddr string) string {
	trimmed := strings.TrimSpace(rawAddr)
	if trimmed == "" {
		return ""
	}
	_, port, err := net.SplitHostPort(trimmed)
	if err == nil {
		return SanitizePort(port)
	}

	if strings.Count(trimmed, ":") <= 1 {
		if idx := strings.LastIndex(trimmed, ":"); idx >= 0 {
			return SanitizePort(trimmed[idx+1:])
		}
	}

	return SanitizePort(trimmed)
}

// SanitizePort validates and normalizes a port string.
func SanitizePort(raw string) string {
	port := strings.TrimSpace(raw)
	if port == "" {
		return ""
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return ""
	}
	if n < 1 || n > 65535 {
		return ""
	}
	return port
}

// ShouldSetLanguageCookie reports whether a cookie should be updated for the request.
func ShouldSetLanguageCookie(r *http.Request, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return false
	}
	if r == nil {
		return true
	}
	cookie, err := r.Cookie(webi18n.LangCookieName)
	if err != nil {
		return true
	}
	return strings.TrimSpace(cookie.Value) != expected
}
