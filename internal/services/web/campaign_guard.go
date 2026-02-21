package web

import (
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// requireCampaignActor resolves and validates the current session member against the
// requested campaign, returning the participant for downstream authorization checks.
func (h *handler) requireCampaignActor(w http.ResponseWriter, r *http.Request, campaignID string) (*statev1.Participant, bool) {
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return nil, false
	}
	if h.campaignAccess == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Campaign unavailable", "campaign access checker is not configured")
		return nil, false
	}

	participant, err := h.campaignParticipant(r.Context(), campaignID, sess)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Campaign unavailable", "failed to verify campaign access")
		return nil, false
	}
	if participant == nil || strings.TrimSpace(participant.GetId()) == "" {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "participant access required")
		return nil, false
	}
	return participant, true
}

// requireCampaignParticipant enforces campaign membership for the current session.
// It writes the response on failure and returns false.
func (h *handler) requireCampaignParticipant(w http.ResponseWriter, r *http.Request, campaignID string) bool {
	_, ok := h.requireCampaignActor(w, r, campaignID)
	return ok
}
