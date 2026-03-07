package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// stableCampaignMutationSurface declares stable session, invite, and game routes.
func stableCampaignMutationSurface() routeSurface {
	return routeSurface{
		id:       "stable-mutations",
		register: registerStableCampaignMutationRoutes,
	}
}

// registerStableCampaignMutationRoutes registers stable mutation and detail routes.
func registerStableCampaignMutationRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionsPattern, h.withCampaignID(h.handleSessions))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionPattern, h.withCampaignAndSessionID(h.handleSessionDetail))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionStartPattern, h.withCampaignID(h.handleSessionStart))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionEndPattern, h.withCampaignID(h.handleSessionEnd))

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignGamePattern, h.withCampaignID(h.handleGame))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignEditPattern, h.withCampaignID(h.handleCampaignUpdate))

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignInvitesPattern, h.withCampaignID(h.handleInvites))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteCreatePattern, h.withCampaignID(h.handleInviteCreate))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteRevokePattern, h.withCampaignID(h.handleInviteRevoke))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignParticipantEditPattern, h.withCampaignAndParticipantID(h.handleParticipantUpdate))
}
