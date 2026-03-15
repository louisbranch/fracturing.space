package campaigns

import (
	"net/http"

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

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignPattern, h.overview.withCampaignID(h.overview.handleOverview))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignPattern, h.overview.handleOverviewMethodNotAllowed)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignEditPattern, h.overview.withCampaignID(h.overview.handleCampaignEdit))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignEditPattern, h.overview.withCampaignID(h.overview.handleCampaignUpdate))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignAIBindingPattern, h.overview.withCampaignID(h.overview.handleCampaignAIBindingPage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignAIBindingPattern, h.overview.withCampaignID(h.overview.handleCampaignAIBinding))
}
