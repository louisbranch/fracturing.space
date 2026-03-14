package publicauth

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// handleLogout handles this route in the module transport layer.
func (h handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionID, hasSession := sessioncookie.Read(r)
	if hasSession && !sessioncookie.AllowsMutationWithPolicy(r, h.requestMeta) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	h.clearSessionCookie(w, r)
	if hasSession {
		_ = h.session.RevokeWebSession(r.Context(), sessionID)
	}
	httpx.WriteRedirect(w, r, routepath.Root)
}

// redirectAuthenticatedToApp centralizes this web behavior in one helper seam.
func (h handlers) redirectAuthenticatedToApp(w http.ResponseWriter, r *http.Request) bool {
	if r == nil {
		return false
	}
	if !h.IsViewerSignedIn(r) {
		return false
	}
	httpx.WriteRedirect(w, r, h.session.ResolvePostAuthRedirect(h.pendingID(r), h.nextPath(r)))
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
