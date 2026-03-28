package campaigns

import (
	"net/http"

	campaigncharacters "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/characters"
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
	campaigncharacters.RegisterStableRoutes(mux, h.characters)
}
