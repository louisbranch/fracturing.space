package web

import (
	"net/http"
)

// handleAppDashboard redirects users to the canonical logged-in shell at root.
func (h *handler) handleAppDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	if sessionFromRequest(r, h.sessions) == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
