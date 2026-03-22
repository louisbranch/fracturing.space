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
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionsPattern, h.sessions.withCampaignID(h.sessions.handleSessions))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionCreatePattern, h.sessions.withCampaignID(h.sessions.handleSessionCreatePage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionCreatePattern, h.sessions.withCampaignID(h.sessions.handleSessionCreate))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionPattern, h.sessions.withCampaignAndSessionID(h.sessions.handleSessionDetail))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionEndPattern, h.sessions.withCampaignID(h.sessions.handleSessionEnd))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignGamePattern, h.sessions.withCampaignID(h.sessions.handleGame))
}
