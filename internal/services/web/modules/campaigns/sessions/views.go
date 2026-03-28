package sessions

import (
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// sessionsView builds the session list page view from workspace state.
func sessionsView(page *campaigndetail.PageContext, campaignID string) campaignrender.SessionsPageView {
	view := campaignrender.SessionsPageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.Sessions = mapSessionsView(page.Sessions)
	return view
}

// sessionsBreadcrumbs returns the root breadcrumb trail for the sessions surface.
func sessionsBreadcrumbs(page *campaigndetail.PageContext) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.sessions.title")},
	}
}

// sessionCreateView builds the session-create page view from readiness state.
func sessionCreateView(page *campaigndetail.PageContext, campaignID string, readiness campaignapp.CampaignSessionReadiness) campaignrender.SessionCreatePageView {
	view := campaignrender.SessionCreatePageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.SessionReadiness = mapSessionReadinessView(readiness)
	return view
}

// sessionCreateBreadcrumbs returns breadcrumbs for the session-create page.
func sessionCreateBreadcrumbs(page *campaigndetail.PageContext, campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.sessions.title"), URL: routepath.AppCampaignSessions(campaignID)},
		{Label: webtemplates.T(page.Loc, "game.sessions.action_new")},
	}
}

// sessionDetailView builds the session-detail page view from workspace sessions.
func sessionDetailView(page *campaigndetail.PageContext, campaignID, sessionID string) campaignrender.SessionDetailPageView {
	view := campaignrender.SessionDetailPageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.SessionID = strings.TrimSpace(sessionID)
	view.Sessions = mapSessionsView(page.Sessions)
	return view
}

// sessionDetailBreadcrumbs returns breadcrumbs for the session-detail page.
func sessionDetailBreadcrumbs(page *campaigndetail.PageContext, campaignID string, view campaignrender.SessionDetailPageView) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.sessions.title"), URL: routepath.AppCampaignSessions(campaignID)},
		{Label: campaignSessionBreadcrumbLabel(page.Loc, view)},
	}
}

// campaignSessionBreadcrumbLabel picks the most specific breadcrumb label for one session.
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
	return webtemplates.T(loc, "game.sessions.title")
}

// mapSessionsView projects app sessions into render rows.
func mapSessionsView(items []campaignapp.CampaignSession) []campaignrender.SessionView {
	result := make([]campaignrender.SessionView, 0, len(items))
	for _, session := range items {
		result = append(result, campaignrender.SessionView{
			ID:        session.ID,
			Name:      session.Name,
			Status:    session.Status,
			StartedAt: session.StartedAt,
			UpdatedAt: session.UpdatedAt,
			EndedAt:   session.EndedAt,
		})
	}
	return result
}

// mapSessionReadinessView projects readiness blockers into render-owned state.
func mapSessionReadinessView(readiness campaignapp.CampaignSessionReadiness) campaignrender.SessionReadinessView {
	result := campaignrender.SessionReadinessView{
		Ready:    readiness.Ready,
		Blockers: make([]campaignrender.SessionReadinessBlockerView, 0, len(readiness.Blockers)),
	}
	for _, blocker := range readiness.Blockers {
		result.Blockers = append(result.Blockers, campaignrender.SessionReadinessBlockerView{
			Code:    blocker.Code,
			Message: blocker.Message,
		})
	}
	return result
}
