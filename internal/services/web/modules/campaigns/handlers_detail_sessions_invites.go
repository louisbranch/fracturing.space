package campaigns

import (
	"net/http"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
)

// handleSessions handles this route in the module transport layer.
func (h handlers) handleSessions(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	readiness, err := h.sessionReads.CampaignSessionReadiness(ctx, campaignID, page.locale)
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
	items, err := h.inviteReads.CampaignInvites(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	var participants []campaignapp.CampaignParticipant
	if page.canManageInvites {
		var err error
		participants, err = h.participantReads.CampaignParticipants(ctx, campaignID)
		if err != nil {
			h.WriteError(w, r, err)
			return
		}
	}
	view := page.invitesView(campaignID, participants, items, r)
	h.writeCampaignDetailPage(w, r, page, campaignID, campaignrender.InvitesFragment(view, page.loc), page.invitesBreadcrumbs()...)
}
