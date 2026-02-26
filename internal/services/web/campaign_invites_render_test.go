package web

import (
	"net/http"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	featureinvites "github.com/louisbranch/fracturing.space/internal/services/web/feature/invites"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/language"
)

func renderAppInvitesPage(w http.ResponseWriter, r *http.Request, invites []*statev1.PendingUserInvite) {
	renderAppInvitesPageWithContext(w, r, webtemplates.PageContext{
		Lang: language.English.String(),
		Loc:  webi18n.Printer(language.English),
	}, invites)
}

func renderAppInvitesPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, invites []*statev1.PendingUserInvite) {
	featureinvites.RenderAppInvitesPage(w, r, page, invites)
}

func renderAppCampaignInvitesPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, invites []*statev1.Invite, canManageInvites bool) {
	campaignfeature.RenderCampaignInvitesPage(w, r, page, campaignID, invites, nil, canManageInvites)
}

func renderAppCampaignsPage(w http.ResponseWriter, r *http.Request, campaigns []*statev1.Campaign) {
	campaignfeature.RenderCampaignsPage(w, r, campaigns)
}

func renderAppCampaignSessionsPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, sessions []*statev1.Session, canManageSessions bool) {
	campaignfeature.RenderCampaignSessionsPage(w, r, page, campaignID, sessions, canManageSessions)
}

func renderAppCampaignSessionDetailPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, session *statev1.Session) {
	campaignfeature.RenderCampaignSessionDetailPage(w, r, page, campaignID, session)
}

func renderAppCampaignParticipantsPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, participants []*statev1.Participant, canManageParticipants bool) {
	campaignfeature.RenderCampaignParticipantsPage(w, r, page, campaignID, participants, canManageParticipants)
}

func renderAppCampaignCharactersPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, characters []*statev1.Character, canManageCharacters bool, controlParticipants []*statev1.Participant) {
	campaignfeature.RenderCampaignCharactersPage(w, r, page, campaignID, characters, canManageCharacters, controlParticipants)
}

func renderAppCampaignCharacterDetailPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, campaignID string, character *statev1.Character) {
	campaignfeature.RenderCampaignCharacterDetailPage(w, r, page, campaignID, character)
}

func renderAppCampaignCreatePage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext) {
	campaignfeature.RenderCampaignCreatePage(w, r, page)
}
