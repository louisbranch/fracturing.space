package admin

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/requestctx"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/shared/authctx"
)

// tokenCookieName is the domain-scoped cookie set by the web login service.
const tokenCookieName = "fs_token"

// AuthConfig holds auth middleware configuration for the admin operator plane.
type AuthConfig struct {
	IntrospectURL  string
	ResourceSecret string
	LoginURL       string
}

// requireAuth wraps next with token-introspection-based authentication.
//
// This keeps admin routes protected by the shared auth service while leaving only
// static assets and login handoff paths unauthenticated.
func requireAuth(next http.Handler, introspector TokenIntrospector, loginURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAuthExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(tokenCookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			http.Redirect(w, r, loginURL, http.StatusFound)
			return
		}

		result, err := introspector.Introspect(r.Context(), cookie.Value)
		if err != nil {
			log.Printf("admin auth introspect error: %v", err)
			http.Redirect(w, r, loginURL, http.StatusFound)
			return
		}
		if !result.Active {
			http.Redirect(w, r, loginURL, http.StatusFound)
			return
		}

		ctx := requestctx.WithUserID(r.Context(), result.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isAuthExempt returns true for paths that should bypass authentication.
func isAuthExempt(path string) bool {
	return strings.HasPrefix(path, routepath.StaticPrefix)
}

// TokenIntrospector validates an OAuth access token via introspection.
type TokenIntrospector = authctx.Introspector

// introspectResponse mirrors the auth service's introspect JSON shape.
type introspectResponse = authctx.IntrospectionResult

// newHTTPIntrospector creates an introspector that POSTs to the given URL.
func newHTTPIntrospector(url, resourceSecret string) TokenIntrospector {
	return authctx.NewHTTPIntrospector(url, resourceSecret, &http.Client{Timeout: 5 * time.Second})
}
