package web

import (
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// handleAppHome redirects authenticated sessions to the app home shell and keeps
// unauthenticated traffic out of the application shell.
func (h *handler) handleAppHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == routepath.AppRootPrefix {
		http.Redirect(w, r, routepath.AppRoot, http.StatusMovedPermanently)
		return
	}
	if r.URL.Path != routepath.AppRoot {
		http.NotFound(w, r)
		return
	}
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
