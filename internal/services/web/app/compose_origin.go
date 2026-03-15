package app

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
)

// requireCookieSessionSameOrigin enforces same-origin proof for cookie-backed
// mutation requests and leaves read requests untouched.
func requireCookieSessionSameOrigin(policy requestmeta.SchemePolicy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMutationMethod(r) || !hasSessionCookie(r) {
				next.ServeHTTP(w, r)
				return
			}
			if !hasSameOriginProof(r, policy) {
				weberror.WriteAppError(w, r, http.StatusForbidden, nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isMutationMethod identifies state-changing HTTP verbs for same-origin
// checks.
func isMutationMethod(r *http.Request) bool {
	if r == nil {
		return false
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// hasSessionCookie reports whether the request carries an authenticated web
// session cookie and therefore requires same-origin mutation proof.
func hasSessionCookie(r *http.Request) bool {
	_, ok := sessioncookie.Read(r)
	return ok
}

// hasSameOriginProof delegates proof validation to shared requestmeta helpers.
func hasSameOriginProof(r *http.Request, policy requestmeta.SchemePolicy) bool {
	return requestmeta.HasSameOriginProofWithPolicy(r, policy)
}
