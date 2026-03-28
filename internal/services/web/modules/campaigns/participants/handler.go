package participants

import (
	"fmt"
	"net/http"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// ServiceConfig groups participant read and mutation app config.
type ServiceConfig struct {
	Read          campaignapp.ParticipantReadServiceConfig
	Mutation      campaignapp.ParticipantMutationServiceConfig
	Authorization campaignapp.AuthorizationGateway
}

// HandlerServices groups participant read and mutation behavior.
type HandlerServices struct {
	reads    campaignapp.CampaignParticipantReadService
	mutation campaignapp.CampaignParticipantMutationService
}

// NewHandlerServices keeps participant transport dependencies owned by the
// participant surface instead of the campaigns root constructor.
func NewHandlerServices(config ServiceConfig) (HandlerServices, error) {
	reads, err := campaignapp.NewParticipantReadService(config.Read, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("participant-reads: %w", err)
	}
	mutation, err := campaignapp.NewParticipantMutationService(config.Mutation, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("participant-mutation: %w", err)
	}
	return HandlerServices{reads: reads, mutation: mutation}, nil
}

// Handler owns participant read and mutation routes.
type Handler struct {
	campaigndetail.Handler
	participants HandlerServices
}

// NewHandler assembles the participant route-owner handler.
func NewHandler(detail campaigndetail.Handler, services HandlerServices) Handler {
	return Handler{
		Handler:      detail,
		participants: services,
	}
}

// HandleParticipants renders the participants detail page.
func (h Handler) HandleParticipants(w http.ResponseWriter, r *http.Request, campaignID string) {
	viewerUserID := h.RequestUserID(r)
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	items, err := h.participants.reads.CampaignParticipants(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := participantsView(page, campaignID, items, viewerUserID, h.Pages.Authorization.RequireManageParticipants(ctx, campaignID) == nil)
	h.WriteCampaignDetailPage(w, r, page, campaignID, campaignrender.ParticipantsFragment(view, page.Loc), participantsBreadcrumbs(page)...)
}

// HandleParticipantCreatePage renders the participant creation page.
func (h Handler) HandleParticipantCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	creator, err := h.participants.reads.CampaignParticipantCreator(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := participantCreateView(page, campaignID, creator)
	h.WriteCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.ParticipantCreateFragment(view, page.Loc),
		participantCreateBreadcrumbs(page, campaignID)...,
	)
}

// HandleParticipantCreate creates a new participant seat.
func (h Handler) HandleParticipantCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	redirectURL := routepath.AppCampaignParticipantCreate(campaignID)
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_participant_create_form", redirectURL) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if _, err := h.participants.mutation.CreateParticipant(ctx, campaignID, parseCreateParticipantInput(r.Form)); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_create_participant", redirectURL)
		return
	}
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_participant_created", routepath.AppCampaignInvites(campaignID))
}

// HandleParticipantEdit renders the participant editor.
func (h Handler) HandleParticipantEdit(w http.ResponseWriter, r *http.Request, campaignID, participantID string) {
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	editor, err := h.participants.reads.CampaignParticipantEditor(ctx, campaignID, participantID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := participantEditView(page, campaignID, participantID, editor)
	h.WriteCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.ParticipantEditFragment(view, page.Loc),
		participantEditBreadcrumbs(page, campaignID)...,
	)
}

// HandleParticipantUpdate updates participant configuration.
func (h Handler) HandleParticipantUpdate(w http.ResponseWriter, r *http.Request, campaignID, participantID string) {
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_participant_update_form", routepath.AppCampaignParticipants(campaignID)) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.participants.mutation.UpdateParticipant(ctx, campaignID, parseUpdateParticipantInput(participantID, r.Form)); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_update_participant", routepath.AppCampaignParticipants(campaignID))
		return
	}
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_participant_updated", routepath.AppCampaignParticipants(campaignID))
}
