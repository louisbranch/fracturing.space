package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// stableCampaignCoreSurface declares stable list/create/workspace read routes.
func stableCampaignCoreSurface() routeSurface {
	return routeSurface{
		id:       "stable-core",
		register: registerStableCampaignCoreRoutes,
	}
}

// registerStableCampaignCoreRoutes registers stable list/create/workspace reads.
func registerStableCampaignCoreRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaigns, h.handleIndex)
	mux.HandleFunc(http.MethodGet+" "+routepath.CampaignsPrefix+"{$}", h.handleIndex)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignsNew, h.handleStartNewCampaign)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignsCreate, h.handleCreateCampaign)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignsCreate, h.handleCreateCampaignSubmit)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignPattern, h.withCampaignID(h.handleOverview))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignPattern, h.handleOverviewMethodNotAllowed)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignEditPattern, h.withCampaignID(h.handleCampaignEdit))

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantsPattern, h.withCampaignID(h.handleParticipants))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantCreatePattern, h.withCampaignID(h.handleParticipantCreatePage))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantEditPattern, h.withCampaignAndParticipantID(h.handleParticipantEdit))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharactersPattern, h.withCampaignID(h.handleCharacters))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterCreatePattern, h.withCampaignID(h.handleCharacterCreatePage))
}
