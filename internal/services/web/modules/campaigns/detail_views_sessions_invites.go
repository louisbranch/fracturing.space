package campaigns

import (
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// sessionsView builds the sessions detail view for one campaign.
func (p *campaignPageContext) sessionsView(campaignID string, readiness campaignapp.CampaignSessionReadiness) campaignrender.SessionsPageView {
	view := campaignrender.SessionsPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.Sessions = mapSessionsView(p.sessions)
	view.SessionReadiness = mapSessionReadinessView(readiness)
	return view
}

// sessionsBreadcrumbs returns breadcrumbs for the sessions list page.
func (p *campaignPageContext) sessionsBreadcrumbs() []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.sessions.title")},
	}
}

// sessionDetailView builds the session-detail view for one campaign.
func (p *campaignPageContext) sessionDetailView(campaignID, sessionID string) campaignrender.SessionDetailPageView {
	view := campaignrender.SessionDetailPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.SessionID = strings.TrimSpace(sessionID)
	view.Sessions = mapSessionsView(p.sessions)
	return view
}

// sessionDetailBreadcrumbs returns breadcrumbs for the selected session page.
func (p *campaignPageContext) sessionDetailBreadcrumbs(campaignID string, view campaignrender.SessionDetailPageView) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.sessions.title"), URL: routepath.AppCampaignSessions(campaignID)},
		{Label: campaignSessionBreadcrumbLabel(p.loc, view)},
	}
}

// campaignSessionBreadcrumbLabel resolves the selected session breadcrumb label.
func campaignSessionBreadcrumbLabel(loc webtemplates.Localizer, view campaignrender.SessionDetailPageView) string {
	selectedSessionID := strings.TrimSpace(view.SessionID)
	if selectedSessionID == "" {
		return webtemplates.T(loc, "game.sessions.title")
	}
	for _, session := range view.Sessions {
		if strings.TrimSpace(session.ID) != selectedSessionID {
			continue
		}
		sessionName := strings.TrimSpace(session.Name)
		if sessionName != "" {
			return sessionName
		}
		break
	}
	return webtemplates.T(loc, "game.sessions.menu.unnamed")
}

// invitesView builds the invites detail view for one campaign.
func (p *campaignPageContext) invitesView(
	campaignID string,
	participants []campaignapp.CampaignParticipant,
	invites []campaignapp.CampaignInvite,
	r *http.Request,
) campaignrender.InvitesPageView {
	view := campaignrender.InvitesPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	if view.CanManageInvites {
		view.InviteSeatOptions = mapInviteSeatOptions(participants, invites)
	}
	view.Invites = mapInvitesView(invites, r)
	return view
}

// invitesBreadcrumbs returns breadcrumbs for the invites page.
func (p *campaignPageContext) invitesBreadcrumbs() []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.campaign_invites.title")},
	}
}
