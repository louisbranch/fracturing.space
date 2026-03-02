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
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionPattern, h.handleSessionDetailRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionStartPattern, h.handleSessionStartRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionEndPattern, h.handleSessionEndRoute)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignGamePattern, h.withCampaignID(h.handleGame))

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignInvitesPattern, h.withCampaignID(h.handleInvites))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteCreatePattern, h.handleInviteCreateRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteRevokePattern, h.handleInviteRevokeRoute)
}
