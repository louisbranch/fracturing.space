package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// stableCampaignCharacterRoutes declares stable character workspace routes.
func stableCampaignCharacterRoutes() routeSurface {
	return routeSurface{
		id:       "stable-characters",
		register: registerStableCampaignCharacterRoutes,
	}
}

// registerStableCampaignCharacterRoutes registers stable character workspace routes.
func registerStableCampaignCharacterRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharactersPattern, h.withCampaignID(h.handleCharacters))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterCreatePattern, h.withCampaignID(h.handleCharacterCreatePage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreatePattern, h.withCampaignID(h.handleCharacterCreate))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterPattern, h.withCampaignAndCharacterID(h.handleCharacterDetail))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterEditPattern, h.withCampaignAndCharacterID(h.handleCharacterEdit))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterEditPattern, h.withCampaignAndCharacterID(h.handleCharacterUpdate))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterControlPattern, h.withCampaignAndCharacterID(h.handleCharacterControlSet))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterControlClaimPattern, h.withCampaignAndCharacterID(h.handleCharacterControlClaim))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterControlReleasePattern, h.withCampaignAndCharacterID(h.handleCharacterControlRelease))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterDeletePattern, h.withCampaignAndCharacterID(h.handleCharacterDelete))
}
