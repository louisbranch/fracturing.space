package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
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
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantsPattern, h.withCampaignID(h.handleParticipants))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantCreatePattern, h.withCampaignID(h.handleParticipantCreatePage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignParticipantCreatePattern, h.withCampaignID(h.handleParticipantCreate))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantEditPattern, h.withCampaignAndParticipantID(h.handleParticipantEdit))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignParticipantEditPattern, h.withCampaignAndParticipantID(h.handleParticipantUpdate))
}
