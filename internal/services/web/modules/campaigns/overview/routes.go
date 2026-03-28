package overview

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// RegisterStableRoutes registers stable campaign overview routes.
func RegisterStableRoutes(mux *http.ServeMux, h Handler) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignPattern, h.WithCampaignID(h.HandleOverview))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignPattern, h.HandleOverviewMethodNotAllowed)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignEditPattern, h.WithCampaignID(h.HandleCampaignEdit))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignEditPattern, h.WithCampaignID(h.HandleCampaignUpdate))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignAIBindingPattern, h.WithCampaignID(h.HandleCampaignAIBindingPage))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignAIBindingPattern, h.WithCampaignID(h.HandleCampaignAIBinding))
}
