package publicauth

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// handleLogout handles this route in the module transport layer.
func (h handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionID, hasSession := sessioncookie.Read(r)
	if hasSession && !h.hasSameOriginProof(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	h.clearSessionCookie(w, r)
	if hasSession {
		_ = h.service.RevokeWebSession(r.Context(), sessionID)
	}
	httpx.WriteRedirect(w, r, routepath.Root)
}

// redirectAuthenticatedToApp centralizes this web behavior in one helper seam.
func (h handlers) redirectAuthenticatedToApp(w http.ResponseWriter, r *http.Request) bool {
	if r == nil {
		return false
	}
	sessionID, ok := sessioncookie.Read(r)
	if !ok {
		return false
	}
	if !h.service.HasValidWebSession(r.Context(), sessionID) {
		return false
	}
	httpx.WriteRedirect(w, r, resolveAppRedirectPath(r.URL.Query().Get("next")))
	return true
}

// writeSessionCookie centralizes this web behavior in one helper seam.
func (h handlers) writeSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string) {
	sessioncookie.WriteWithPolicy(w, r, sessionID, h.requestMeta)
}

// clearSessionCookie centralizes this web behavior in one helper seam.
func (h handlers) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	sessioncookie.ClearWithPolicy(w, r, h.requestMeta)
}

// hasSameOriginProof reports whether this package condition is satisfied.
func (h handlers) hasSameOriginProof(r *http.Request) bool {
	return requestmeta.HasSameOriginProofWithPolicy(r, h.requestMeta)
}

// resolveAppRedirectPath resolves request-scoped values needed by this package.
func resolveAppRedirectPath(raw string) string {
	next := strings.TrimSpace(raw)
	if next == "" {
		return routepath.AppDashboard
	}
	parsed, err := url.Parse(next)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.Opaque != "" {
		return routepath.AppDashboard
	}
	rawPath := strings.TrimSpace(parsed.EscapedPath())
	if hasEncodedSlash(rawPath) {
		return routepath.AppDashboard
	}
	decodedPath, err := url.PathUnescape(strings.TrimSpace(parsed.Path))
	if err != nil {
		return routepath.AppDashboard
	}
	if hasDotSegment(decodedPath) {
		return routepath.AppDashboard
	}
	canonicalPath := path.Clean(decodedPath)
	if strings.TrimSpace(canonicalPath) == "." {
		canonicalPath = "/"
	}
	canonicalPath = ensureLeadingSlash(canonicalPath)
	if !strings.HasPrefix(canonicalPath, routepath.AppPrefix) {
		return routepath.AppDashboard
	}
	if canonicalPath == routepath.AppPrefix {
		return routepath.AppDashboard
	}
	if parsed.RawQuery != "" {
		return canonicalPath + "?" + parsed.RawQuery
	}
	return canonicalPath
}

// hasDotSegment reports whether this package condition is satisfied.
func hasDotSegment(rawPath string) bool {
	for _, part := range strings.Split(rawPath, "/") {
		if part == "." || part == ".." {
			return true
		}
	}
	return false
}

// hasEncodedSlash reports whether this package condition is satisfied.
func hasEncodedSlash(rawPath string) bool {
	lower := strings.ToLower(rawPath)
	return strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c")
}

// ensureLeadingSlash centralizes this web behavior in one helper seam.
func ensureLeadingSlash(pathValue string) string {
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		return "/"
	}
	if strings.HasPrefix(pathValue, "/") {
		return pathValue
	}
	return "/" + pathValue
}
