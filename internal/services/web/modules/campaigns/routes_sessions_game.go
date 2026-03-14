package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
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
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionsPattern, h.withCampaignID(h.handleSessions))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionPattern, h.withCampaignAndSessionID(h.handleSessionDetail))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionStartPattern, h.withCampaignID(h.handleSessionStart))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionEndPattern, h.withCampaignID(h.handleSessionEnd))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignGamePattern, h.withCampaignID(h.handleGame))
}
