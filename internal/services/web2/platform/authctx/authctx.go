// Package authctx provides web2 authentication seams.
package authctx

import (
	"context"
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/sessioncookie"
)

// IsAuthenticated reports whether the current request should access protected routes.
type IsAuthenticated func(*http.Request) bool

// HeaderOrCookieAuth returns a cookie-only auth strategy for protected routes.
//
// This seam intentionally rejects header-only identities.
func HeaderOrCookieAuth() IsAuthenticated {
	return func(r *http.Request) bool {
		if r == nil {
			return false
		}
		_, ok := sessioncookie.Read(r)
		return ok
	}
}

// HeaderOrValidatedSessionAuth validates session cookies and rejects header-only identities.
func HeaderOrValidatedSessionAuth(validate func(context.Context, string) bool) IsAuthenticated {
	validated := ValidatedSessionAuth(validate)
	return func(r *http.Request) bool {
		if r == nil {
			return false
		}
		return validated(r)
	}
}

// ValidatedSessionAuth authenticates requests only through validated session cookies.
func ValidatedSessionAuth(validate func(context.Context, string) bool) IsAuthenticated {
	return func(r *http.Request) bool {
		if r == nil || validate == nil {
			return false
		}
		sessionID, ok := sessioncookie.Read(r)
		if !ok {
			return false
		}
		return validate(r.Context(), sessionID)
	}
}
