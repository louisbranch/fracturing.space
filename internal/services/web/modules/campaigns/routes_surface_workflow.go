package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// stableCampaignWorkflowSurface declares stable character workflow routes.
func stableCampaignWorkflowSurface() routeSurface {
	return routeSurface{
		id:       "stable-workflow",
		register: registerStableCampaignWorkflowRoutes,
	}
}

// registerStableCampaignWorkflowRoutes registers stable character workflow routes.
func registerStableCampaignWorkflowRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterPattern, h.handleCharacterDetailRoute)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterCreationPattern, h.handleCharacterCreationPageRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterPattern, h.handleCharacterUpdateRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreatePattern, h.handleCharacterCreateRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationStepPattern, h.handleCharacterCreationStepRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationResetPattern, h.handleCharacterCreationResetRoute)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignRestPattern, h.WriteNotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignRestPattern, h.WriteNotFound)
}
