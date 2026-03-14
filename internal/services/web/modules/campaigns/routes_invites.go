package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
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
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignInvitesPattern, h.withCampaignID(h.handleInvites))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteCreatePattern, h.withCampaignID(h.handleInviteCreate))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteRevokePattern, h.withCampaignID(h.handleInviteRevoke))
}
