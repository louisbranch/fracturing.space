package web

import (
	"net/http"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	websupport "github.com/louisbranch/fracturing.space/internal/services/web/support"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func buildCampaignFeatureRenderDependencies(h *handler, d *campaignfeature.AppCampaignDependencies) {
	d.LocalizeError = localizeHTTPError
	d.RenderErrorPage = func(w http.ResponseWriter, r *http.Request, status int, title string, message string) {
		h.renderErrorPage(w, r, status, title, message)
	}
	d.GRPCErrorStatus = websupport.GRPCErrorHTTPStatus
	d.RenderCampaignsListPage = func(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaigns []*statev1.Campaign) {
		campaignfeature.RenderCampaignsListPageWithConfig(w, r, page, h.config.AssetBaseURL, campaigns)
	}
	d.RenderCampaignCreatePage = func(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext) {
		campaignfeature.RenderCampaignCreatePage(w, r, page)
	}
	d.RenderCampaignPage = func(w http.ResponseWriter, r *http.Request, campaignID string) {
		h.renderCampaignPage(w, r, campaignID)
	}
	d.RenderCampaignSessionsPage = func(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, sessions []*statev1.Session, canManage bool) {
		campaignfeature.RenderCampaignSessionsPage(w, r, page, campaignID, sessions, canManage)
	}
	d.RenderCampaignSessionDetailPage = func(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, session *statev1.Session) {
		campaignfeature.RenderCampaignSessionDetailPage(w, r, page, campaignID, session)
	}
	d.RenderCampaignParticipantsPage = func(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, participants []*statev1.Participant, canManage bool) {
		campaignfeature.RenderCampaignParticipantsPage(w, r, page, campaignID, participants, canManage)
	}
	d.RenderCampaignCharactersPage = func(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, characters []*statev1.Character, canManage bool, controlParticipants []*statev1.Participant) {
		campaignfeature.RenderCampaignCharactersPage(w, r, page, campaignID, characters, canManage, controlParticipants)
	}
	d.RenderCampaignCharacterDetailPage = func(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, character *statev1.Character) {
		campaignfeature.RenderCampaignCharacterDetailPage(w, r, page, campaignID, character)
	}
	d.RenderCampaignInvitesPage = func(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, contacts []webtemplates.CampaignInviteContactOption, canManage bool) {
		campaignfeature.RenderCampaignInvitesPageWithContext(w, r, page, campaignID, invites, contacts, canManage, webtemplates.CampaignInviteVerification{})
	}
	d.RenderCampaignInvitesVerificationPage = func(w http.ResponseWriter, r *http.Request, campaignID string, userID string, canManageInvites bool, verification webtemplates.CampaignInviteVerification) {
		campaignfeature.HandleAppCampaignInvitesVerification(*d, w, r, campaignID, userID, canManageInvites, verification)
	}
}
