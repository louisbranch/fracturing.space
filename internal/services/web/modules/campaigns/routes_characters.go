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
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharactersPattern, h.characters.withCampaignID(h.characters.handleCharacters))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterCreatePattern, h.characters.withCampaignID(h.characters.handleCharacterCreatePage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreatePattern, h.characters.withCampaignID(h.characters.handleCharacterCreate))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterPattern, h.characters.withCampaignAndCharacterID(h.characters.handleCharacterDetail))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterEditPattern, h.characters.withCampaignAndCharacterID(h.characters.handleCharacterEdit))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterEditPattern, h.characters.withCampaignAndCharacterID(h.characters.handleCharacterUpdate))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterControlPattern, h.characters.withCampaignAndCharacterID(h.characters.handleCharacterControlSet))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterControlClaimPattern, h.characters.withCampaignAndCharacterID(h.characters.handleCharacterControlClaim))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterControlReleasePattern, h.characters.withCampaignAndCharacterID(h.characters.handleCharacterControlRelease))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterDeletePattern, h.characters.withCampaignAndCharacterID(h.characters.handleCharacterDelete))
}
