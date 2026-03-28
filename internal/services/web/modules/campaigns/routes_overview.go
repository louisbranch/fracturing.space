package campaigns

import (
	"net/http"

	campaignoverview "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/overview"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// stableCampaignOverviewRoutes declares stable campaign overview routes.
func stableCampaignOverviewRoutes() routeSurface {
	return routeSurface{
		id:       "stable-overview",
		register: registerStableCampaignOverviewRoutes,
	}
}

// registerStableCampaignOverviewRoutes registers stable campaign overview routes.
func registerStableCampaignOverviewRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaigns, h.catalog.handleIndex)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignsNew, h.catalog.handleStartNewCampaign)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignsCreate, h.catalog.handleCreateCampaign)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignsCreate, h.catalog.handleCreateCampaignSubmit)
	campaignoverview.RegisterStableRoutes(mux, h.overview)
}
