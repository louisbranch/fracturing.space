package campaigns

import (
	"net/http"

	campaignsessions "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/sessions"
)

// stableCampaignSessionGameRoutes declares stable session and game routes.
func stableCampaignSessionGameRoutes() routeSurface {
	return routeSurface{
		id:       "stable-sessions-game",
		register: registerStableCampaignSessionGameRoutes,
	}
}

// registerStableCampaignSessionGameRoutes registers stable session and game routes.
func registerStableCampaignSessionGameRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	campaignsessions.RegisterStableRoutes(mux, h.sessions)
}
