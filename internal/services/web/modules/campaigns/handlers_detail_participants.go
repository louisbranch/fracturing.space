package campaigns

import (
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handleParticipants handles this route in the module transport layer.
func (h handlers) handleParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	items, err := h.service.CampaignParticipants(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.detailView(campaignID, markerParticipants)
	if err := h.service.RequireManageParticipants(ctx, campaignID); err == nil {
		view.CanManageParticipants = true
	}
	view.Participants = mapParticipantsView(items)
	h.writeCampaignDetailPage(w, r, page, campaignID, view, sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.participants.title")})
}

// handleParticipantCreatePage handles this route in the module transport layer.
func (h handlers) handleParticipantCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	creator, err := h.service.CampaignParticipantCreator(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.detailView(campaignID, markerParticipantCreate)
	view.CanManageParticipants = true
	view.ParticipantCreator = mapParticipantCreatorView(creator)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		view,
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.participants.title"), URL: routepath.AppCampaignParticipants(campaignID)},
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.participants.action_add")},
	)
}

// handleParticipantEdit handles this route in the module transport layer.
func (h handlers) handleParticipantEdit(w http.ResponseWriter, r *http.Request, campaignID, participantID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	editor, err := h.service.CampaignParticipantEditor(ctx, campaignID, participantID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.detailView(campaignID, markerParticipantEdit)
	view.ParticipantID = editor.Participant.ID
	view.ParticipantEditor = mapParticipantEditorView(editor)
	if strings.TrimSpace(view.ParticipantID) == "" {
		view.ParticipantID = participantID
	}
	if strings.EqualFold(strings.TrimSpace(editor.Participant.Controller), "AI") {
		aiBinding, err := h.service.CampaignAIBindingEditor(ctx, campaignID, page.workspace.AIAgentID)
		if err != nil {
			h.WriteError(w, r, err)
			return
		}
		view.AIBindingEditor = mapAIBindingEditorView(aiBinding)
	}
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		view,
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.participants.title"), URL: routepath.AppCampaignParticipants(campaignID)},
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.participants.action_edit")},
	)
}
