package campaigns

import (
	"net/http"

	campaigninvites "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/invites"
)

// stableCampaignInviteRoutes declares stable invite routes.
func stableCampaignInviteRoutes() routeSurface {
	return routeSurface{
		id:       "stable-invites",
		register: registerStableCampaignInviteRoutes,
	}
}

// registerStableCampaignInviteRoutes registers stable invite routes.
func registerStableCampaignInviteRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	campaigninvites.RegisterStableRoutes(mux, h.invites)
}
