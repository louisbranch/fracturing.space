package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// stableCampaignStarterRoutes declares protected starter preview and launch routes.
func stableCampaignStarterRoutes() routeSurface {
	return routeSurface{
		id:       "stable-starters",
		register: registerStableCampaignStarterRoutes,
	}
}

// registerStableCampaignStarterRoutes registers the protected starter preview and launch routes.
func registerStableCampaignStarterRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil || h.starters.starters == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignStarterPattern, h.handleStarterPreview)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignStarterLaunchPattern, h.handleStarterLaunch)
}
