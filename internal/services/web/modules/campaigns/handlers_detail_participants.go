package campaigns

import (
	"net/http"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
)

// participantHandlerServices groups participant read and mutation behavior.
type participantHandlerServices struct {
	reads    campaignapp.CampaignParticipantReadService
	mutation campaignapp.CampaignParticipantMutationService
}

// participantHandlers owns participant read and mutation routes.
type participantHandlers struct {
	campaignDetailHandlers
	participants participantHandlerServices
}

// newParticipantHandlerServices keeps participant transport dependencies owned
// by the participant surface instead of the root constructor.
func newParticipantHandlerServices(config participantServiceConfig) participantHandlerServices {
	reads := campaignapp.NewParticipantReadService(config.Read, config.Authorization)
	return participantHandlerServices{
		reads:    reads,
		mutation: campaignapp.NewParticipantMutationService(config.Mutation, config.Authorization),
	}
}

// newParticipantHandlers assembles the participant route-owner handler.
func newParticipantHandlers(detail campaignDetailHandlers, services participantHandlerServices) participantHandlers {
	return participantHandlers{
		campaignDetailHandlers: detail,
		participants:           services,
	}
}

// missingParticipantHandlerServices reports which participant route services
// are absent before the participant surface is mounted.
func missingParticipantHandlerServices(services participantHandlerServices) []string {
	missing := []string{}
	if services.reads == nil {
		missing = append(missing, "participant-reads")
	}
	if services.mutation == nil {
		missing = append(missing, "participant-mutation")
	}
	return missing
}

// handleParticipants handles this route in the module transport layer.
func (h participantHandlers) handleParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
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
func (h participantHandlers) handleParticipantCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
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
func (h participantHandlers) handleParticipantEdit(w http.ResponseWriter, r *http.Request, campaignID, participantID string) {
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
