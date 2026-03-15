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
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaigns, h.handleIndex)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignsNew, h.handleStartNewCampaign)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignsCreate, h.handleCreateCampaign)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignsCreate, h.handleCreateCampaignSubmit)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignPattern, h.withCampaignID(h.handleOverview))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignPattern, h.handleOverviewMethodNotAllowed)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignEditPattern, h.withCampaignID(h.handleCampaignEdit))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignEditPattern, h.withCampaignID(h.handleCampaignUpdate))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignAIBindingPattern, h.withCampaignID(h.handleCampaignAIBindingPage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignAIBindingPattern, h.withCampaignID(h.handleCampaignAIBinding))
}
