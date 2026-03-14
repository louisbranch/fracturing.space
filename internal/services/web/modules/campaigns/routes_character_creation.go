package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// stableCampaignCharacterCreationRoutes declares stable character-creation routes.
func stableCampaignCharacterCreationRoutes() routeSurface {
	return routeSurface{
		id:       "stable-character-creation",
		register: registerStableCampaignCharacterCreationRoutes,
	}
}

// registerStableCampaignCharacterCreationRoutes registers stable character-creation routes.
func registerStableCampaignCharacterCreationRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterCreationPattern, h.withCampaignAndCharacterID(h.handleCharacterCreationPage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationStepPattern, h.withCampaignAndCharacterID(h.handleCharacterCreationStep))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationResetPattern, h.withCampaignAndCharacterID(h.handleCharacterCreationReset))

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignRestPattern, h.WriteNotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignRestPattern, h.WriteNotFound)
}
