package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func registerStableRoutes(mux *http.ServeMux, h handlers) {
	registerCommonRoutes(mux, h)
	registerStableWorkflowRoutes(mux, h)
}

func registerExperimentalRoutes(mux *http.ServeMux, h handlers) {
	registerCommonRoutes(mux, h)
	registerStableWorkflowRoutes(mux, h)
	registerExperimentalRoutesForCampaigns(mux, h)
}

func registerCommonRoutes(mux *http.ServeMux, h handlers) {
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

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantsPattern, h.withCampaignID(h.handleParticipants))

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharactersPattern, h.withCampaignID(h.handleCharacters))
}

func registerStableWorkflowRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterPattern, h.handleCharacterDetailRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreatePattern, h.handleCharacterCreateRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationStepPattern, h.handleCharacterCreationStepRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignCharacterCreationResetPattern, h.handleCharacterCreationResetRoute)

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignRestPattern, h.WriteNotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignRestPattern, h.WriteNotFound)
}

// registerExperimentalRoutesForCampaigns adds high-churn routes that are not
// yet promoted to stable surface ownership.
func registerExperimentalRoutesForCampaigns(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionsPattern, h.withCampaignID(h.handleSessions))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionPattern, h.handleSessionDetailRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionStartPattern, h.handleSessionStartRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignSessionEndPattern, h.handleSessionEndRoute)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignGamePattern, h.withCampaignID(h.handleGame))
	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignInvitesPattern, h.withCampaignID(h.handleInvites))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteCreatePattern, h.handleInviteCreateRoute)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppCampaignInviteRevokePattern, h.handleInviteRevokeRoute)
}
