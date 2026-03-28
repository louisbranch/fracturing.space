package campaigns

import (
	"net/http"

	campaignparticipants "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/participants"
)

// stableCampaignParticipantRoutes declares stable participant routes.
func stableCampaignParticipantRoutes() routeSurface {
	return routeSurface{
		id:       "stable-participants",
		register: registerStableCampaignParticipantRoutes,
	}
}

// registerStableCampaignParticipantRoutes registers stable participant routes.
func registerStableCampaignParticipantRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	campaignparticipants.RegisterStableRoutes(mux, h.participants)
}
