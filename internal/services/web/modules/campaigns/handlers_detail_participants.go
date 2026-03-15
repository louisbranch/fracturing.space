package campaigns

import (
	"net/http"

	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
)

// handleParticipants handles this route in the module transport layer.
func (h handlers) handleParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
	viewerUserID := h.RequestUserID(r)
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	items, err := h.participants.reads.CampaignParticipants(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.participantsView(campaignID, items, viewerUserID, h.pages.authorization.RequireManageParticipants(ctx, campaignID) == nil)
	h.writeCampaignDetailPage(w, r, page, campaignID, campaignrender.ParticipantsFragment(view, page.loc), page.participantsBreadcrumbs()...)
}

// handleParticipantCreatePage handles this route in the module transport layer.
func (h handlers) handleParticipantCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	creator, err := h.participants.reads.CampaignParticipantCreator(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.participantCreateView(campaignID, creator)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.ParticipantCreateFragment(view, page.loc),
		page.participantCreateBreadcrumbs(campaignID)...,
	)
}

// handleParticipantEdit handles this route in the module transport layer.
func (h handlers) handleParticipantEdit(w http.ResponseWriter, r *http.Request, campaignID, participantID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	editor, err := h.participants.reads.CampaignParticipantEditor(ctx, campaignID, participantID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.participantEditView(campaignID, participantID, editor)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.ParticipantEditFragment(view, page.loc),
		page.participantEditBreadcrumbs(campaignID)...,
	)
}
