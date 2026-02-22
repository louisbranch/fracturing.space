package web

import (
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// handleAppDashboard redirects users to the canonical logged-in campaigns workspace.
func (h *handler) handleAppDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	if sessionFromRequest(r, h.sessions) == nil {
		http.Redirect(w, r, routepath.AuthLogin, http.StatusFound)
		return
	}

	http.Redirect(w, r, routepath.AppCampaigns, http.StatusFound)
}
