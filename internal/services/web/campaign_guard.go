package web

import "net/http"

// requireCampaignParticipant enforces campaign membership for the current session.
// It writes the response on failure and returns false.
func (h *handler) requireCampaignParticipant(w http.ResponseWriter, r *http.Request, campaignID string) bool {
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return false
	}
	if h.campaignAccess == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Campaign unavailable", "campaign access checker is not configured")
		return false
	}

	allowed, err := h.campaignAccess.IsCampaignParticipant(r.Context(), campaignID, sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Campaign unavailable", "failed to verify campaign access")
		return false
	}
	if !allowed {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "participant access required")
		return false
	}
	return true
}
