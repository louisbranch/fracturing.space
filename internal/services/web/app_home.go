package web

import "net/http"

// handleAppHome redirects authenticated sessions to campaign workspace and keeps
// unauthenticated traffic out of the application shell.
func (h *handler) handleAppHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if sessionFromRequest(r, h.sessions) == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/app/campaigns", http.StatusFound)
}
