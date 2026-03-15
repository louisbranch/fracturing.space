package campaigns

import (
	"net/http"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// handleSessions handles this route in the module transport layer.
func (h handlers) handleSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	readiness, err := h.pages.sessionReads.CampaignSessionReadiness(ctx, campaignID, page.locale)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.sessionsView(campaignID, readiness)
	h.writeCampaignDetailPage(w, r, page, campaignID, campaignrender.SessionsFragment(view, page.loc), page.sessionsBreadcrumbs()...)
}

// handleSessionDetail handles this route in the module transport layer.
func (h handlers) handleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID, sessionID string) {
	_, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if !campaignHasSession(page.sessions, sessionID) {
		h.WriteError(w, r, apperrors.E(apperrors.KindNotFound, "session not found"))
		return
	}
	view := page.sessionDetailView(campaignID, sessionID)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.SessionDetailFragment(view, page.loc),
		page.sessionDetailBreadcrumbs(campaignID, view)...,
	)
}

// handleInvites handles this route in the module transport layer.
func (h handlers) handleInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	items, err := h.invites.reads.CampaignInvites(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	var participants []campaignapp.CampaignParticipant
	if page.canManageInvites {
		var err error
		participants, err = h.invites.participantReads.CampaignParticipants(ctx, campaignID)
		if err != nil {
			h.WriteError(w, r, err)
			return
		}
	}
	view := page.invitesView(campaignID, participants, items, r)
	h.writeCampaignDetailPage(w, r, page, campaignID, campaignrender.InvitesFragment(view, page.loc), page.invitesBreadcrumbs()...)
}

// campaignHasSession ensures detail routes render only stored campaign sessions,
// never synthetic labels derived from route parameters.
func campaignHasSession(sessions []campaignapp.CampaignSession, sessionID string) bool {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false
	}
	for _, session := range sessions {
		if strings.TrimSpace(session.ID) == sessionID {
			return true
		}
	}
	return false
}
