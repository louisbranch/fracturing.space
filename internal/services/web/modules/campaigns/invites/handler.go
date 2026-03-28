package invites

import (
	"fmt"
	"net/http"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// ServiceConfig groups invite read, mutation, and search-adjacent app config.
type ServiceConfig struct {
	Read            campaignapp.InviteReadServiceConfig
	Mutation        campaignapp.InviteMutationServiceConfig
	ParticipantRead campaignapp.ParticipantReadServiceConfig
	Authorization   campaignapp.AuthorizationGateway
}

// HandlerServices groups invite reads, mutations, and recipient lookup
// behavior.
type HandlerServices struct {
	reads            campaignapp.CampaignInviteReadService
	mutation         campaignapp.CampaignInviteMutationService
	participantReads campaignapp.CampaignParticipantReadService
}

// NewHandlerServices keeps invite transport dependencies owned by the invite
// surface instead of the campaigns root constructor.
func NewHandlerServices(config ServiceConfig) (HandlerServices, error) {
	reads, err := campaignapp.NewInviteReadService(config.Read, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("invite-reads: %w", err)
	}
	mutation, err := campaignapp.NewInviteMutationService(config.Mutation, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("invite-mutation: %w", err)
	}
	participantReads, err := campaignapp.NewParticipantReadService(config.ParticipantRead, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("invite participant-reads: %w", err)
	}
	return HandlerServices{reads: reads, mutation: mutation, participantReads: participantReads}, nil
}

// Handler owns invite read, search, and mutation routes.
type Handler struct {
	campaigndetail.Handler
	invites HandlerServices
}

// NewHandler assembles the invite route-owner handler.
func NewHandler(detail campaigndetail.Handler, services HandlerServices) Handler {
	return Handler{
		Handler: detail,
		invites: services,
	}
}

// HandleInvites renders the invite management page.
func (h Handler) HandleInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	items, err := h.invites.reads.CampaignInvites(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := invitesView(page, campaignID, items, r)
	h.WriteCampaignDetailPage(w, r, page, campaignID, campaignrender.InvitesFragment(view, page.Loc), invitesBreadcrumbs(page)...)
}

// HandleInviteCreatePage renders the dedicated invite creation page.
func (h Handler) HandleInviteCreatePage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if err := h.Pages.Authorization.RequireManageInvites(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}

	participants, err := h.invites.participantReads.CampaignParticipants(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	items, err := h.invites.reads.CampaignInvites(ctx, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	view := inviteCreateView(page, campaignID, participants, items)
	h.WriteCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.InviteCreateFragment(view, page.Loc),
		inviteCreateBreadcrumbs(page, campaignID)...,
	)
}

// HandleInviteCreate creates a new invite.
func (h Handler) HandleInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	redirectURL := routepath.AppCampaignInviteCreate(campaignID)
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_invite_create_form", redirectURL) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	input := parseCreateInviteInput(r.Form)
	if err := h.invites.mutation.CreateInvite(ctx, campaignID, input); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_create_invite", redirectURL)
		return
	}
	h.Sync().InviteChanged(ctx, []string{userID}, campaignID)
	noticeKey := "web.campaigns.notice_invite_created"
	if input.RecipientUsername != "" {
		noticeKey = "web.campaigns.notice_invite_sent"
	}
	h.WriteMutationSuccess(w, r, noticeKey, routepath.AppCampaignInvites(campaignID))
}

// HandleInviteRevoke revokes an invite.
func (h Handler) HandleInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_invite_revoke_form", routepath.AppCampaignInvites(campaignID)) {
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	if err := h.invites.mutation.RevokeInvite(ctx, campaignID, parseRevokeInviteInput(r.Form)); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_revoke_invite", routepath.AppCampaignInvites(campaignID))
		return
	}
	h.Sync().InviteChanged(ctx, []string{userID}, campaignID)
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_invite_revoked", routepath.AppCampaignInvites(campaignID))
}
