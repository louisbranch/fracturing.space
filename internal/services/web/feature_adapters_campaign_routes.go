package web

import (
	"net/http"

	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
)

func (h *handler) appCampaignRouteHandlers() campaignfeature.Handlers {
	dependencies := h.appCampaignRouteDependencies()
	return campaignfeature.Handlers{
		Campaigns: func(w http.ResponseWriter, r *http.Request) {
			campaignfeature.HandleAppCampaigns(dependencies.appCampaignDependencies, w, r)
		},
		CampaignCreate: func(w http.ResponseWriter, r *http.Request) {
			campaignfeature.HandleAppCampaignCreate(dependencies.appCampaignDependencies, w, r)
		},
		CampaignOverview: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignOverview(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignSessions: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignSessions(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignSessionStart: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignSessionStart(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignSessionEnd: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignSessionEnd(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignSessionDetail: func(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string) {
			campaignfeature.HandleAppCampaignSessionDetail(dependencies.appCampaignDependencies, w, r, campaignID, sessionID)
		},
		CampaignParticipants: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignParticipants(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignParticipantUpdate: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignParticipantUpdate(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignCharacters: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignCharacters(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignCharacterCreate: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignCharacterCreate(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignCharacterUpdate: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignCharacterUpdate(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignCharacterControl: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignCharacterControl(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignCharacterDetail: func(w http.ResponseWriter, r *http.Request, campaignID string, characterID string) {
			campaignfeature.HandleAppCampaignCharacterDetail(dependencies.appCampaignDependencies, w, r, campaignID, characterID)
		},
		CampaignInvites: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignInvites(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignInviteCreate: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignInviteCreate(dependencies.appCampaignDependencies, w, r, campaignID)
		},
		CampaignInviteRevoke: func(w http.ResponseWriter, r *http.Request, campaignID string) {
			campaignfeature.HandleAppCampaignInviteRevoke(dependencies.appCampaignDependencies, w, r, campaignID)
		},
	}
}
