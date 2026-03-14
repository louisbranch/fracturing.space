package campaigns

import (
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handleSessions handles this route in the module transport layer.
func (h handlers) handleSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	readiness, err := h.service.CampaignSessionReadiness(ctx, campaignID, page.locale)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.detailView(campaignID, markerSessions)
	view.Sessions = mapSessionsView(page.sessions)
	view.SessionReadiness = mapSessionReadinessView(readiness)
	h.writeCampaignDetailPage(w, r, page, campaignID, view, sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.sessions.title")})
}

// handleSessionDetail handles this route in the module transport layer.
func (h handlers) handleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID, sessionID string) {
	_, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	view := page.detailView(campaignID, markerSessionDetail)
	view.SessionID = sessionID
	view.Sessions = mapSessionsView(page.sessions)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		view,
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.sessions.title"), URL: routepath.AppCampaignSessions(campaignID)},
		sharedtemplates.BreadcrumbItem{Label: campaignSessionBreadcrumbLabel(page.loc, view)},
	)
}

// campaignSessionBreadcrumbLabel resolves the selected session breadcrumb label.
func campaignSessionBreadcrumbLabel(loc webtemplates.Localizer, view webtemplates.CampaignDetailView) string {
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

// handleInvites handles this route in the module transport layer.
func (h handlers) handleInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	items, err := h.service.CampaignInvites(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.detailView(campaignID, markerInvites)
	if err := h.service.RequireManageInvites(ctx, campaignID); err == nil {
		view.CanManageInvites = true
		participants, err := h.service.CampaignParticipants(ctx, campaignID)
		if err != nil {
			h.WriteError(w, r, err)
			return
		}
		view.InviteSeatOptions = mapInviteSeatOptions(participants, items)
	}
	view.Invites = mapInvitesView(items)
	h.writeCampaignDetailPage(w, r, page, campaignID, view, sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.campaign_invites.title")})
}
