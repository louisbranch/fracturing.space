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
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterCreationPattern, h.creation.withCampaignAndCharacterID(h.creation.handleCharacterCreationPage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationStepPattern, h.creation.withCampaignAndCharacterID(h.creation.handleCharacterCreationStep))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationResetPattern, h.creation.withCampaignAndCharacterID(h.creation.handleCharacterCreationReset))

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignRestPattern, h.creation.WriteNotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignRestPattern, h.creation.WriteNotFound)
}
