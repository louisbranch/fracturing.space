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
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterEditPattern, h.withCampaignAndCharacterID(h.handleCharacterEdit))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterPattern, h.withCampaignAndCharacterID(h.handleCharacterDetail))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterCreationPattern, h.withCampaignAndCharacterID(h.handleCharacterCreationPage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterEditPattern, h.withCampaignAndCharacterID(h.handleCharacterUpdate))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreatePattern, h.withCampaignID(h.handleCharacterCreate))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationStepPattern, h.withCampaignAndCharacterID(h.handleCharacterCreationStep))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationResetPattern, h.withCampaignAndCharacterID(h.handleCharacterCreationReset))

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignRestPattern, h.WriteNotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignRestPattern, h.WriteNotFound)
}
