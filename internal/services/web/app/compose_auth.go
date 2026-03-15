package app

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

const defaultLoginPath = routepath.Login

// requireAuth wraps protected module handlers with session-backed auth checks
// so every protected mount shares the same login continuation behavior.
func requireAuth(authenticated func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			return http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authenticated(r) {
				httpx.WriteRedirect(w, r, loginRedirectPath(r))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// loginRedirectPath preserves the blocked destination so auth can resume the
// original protected request after the user signs in.
func loginRedirectPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return defaultLoginPath
	}
	nextPath := strings.TrimSpace(r.URL.RequestURI())
	if nextPath == "" {
		return defaultLoginPath
	}
	values := url.Values{}
	values.Set("next", nextPath)
	return defaultLoginPath + "?" + values.Encode()
}

// wrapProtectedModule composes auth and same-origin protections for protected
// modules so each protected mount receives identical guardrails.
func wrapProtectedModule(authenticated func(*http.Request) bool, policy requestmeta.SchemePolicy) func(http.Handler) http.Handler {
	authWrap := requireAuth(authenticated)
	sameOriginWrap := requireCookieSessionSameOrigin(policy)
	return func(next http.Handler) http.Handler {
		return authWrap(sameOriginWrap(next))
	}
}
