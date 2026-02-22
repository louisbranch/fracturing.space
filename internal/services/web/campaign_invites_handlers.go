package web

import (
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func (h *handler) handleAppCampaignInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignInvites lists invites for a campaign and relies on
	// game-service policy to enforce manager/owner-only visibility.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	readCtx, userID, ok := h.campaignReadContext(w, r, "Invites unavailable")
	if !ok {
		return
	}
	readReq := r.WithContext(readCtx)
	if h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invites unavailable", "campaign invite service is not configured")
		return
	}

	var invites []*statev1.Invite
	if cachedInvites, ok := h.cachedCampaignInvites(readCtx, campaignID, userID); ok {
		invites = cachedInvites
	} else {
		resp, err := h.inviteClient.ListInvites(readCtx, &statev1.ListInvitesRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invites unavailable", "failed to list campaign invites")
			return
		}
		invites = resp.GetInvites()
		h.setCampaignInvitesCache(readCtx, campaignID, userID, invites)
	}

	contactOptions := h.listInviteContactOptions(readCtx, campaignID, userID, invites)
	renderAppCampaignInvitesPageWithContextAndContacts(
		w,
		readReq,
		h.pageContextForCampaign(w, readReq, campaignID),
		campaignID,
		invites,
		contactOptions,
		true,
	)
}

func (h *handler) handleAppCampaignInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignInviteCreate creates a player invitation and binds it to
	// the selected target participant.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	inviteActor := h.campaignInviteActorFromParticipant(actor)
	if inviteActor == nil || !inviteActor.canManageInvites {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for invite action")
		return
	}
	if h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invite action unavailable", "campaign invite service is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "failed to parse invite create form")
		return
	}
	targetParticipantID := strings.TrimSpace(r.FormValue("participant_id"))
	if targetParticipantID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "participant id is required")
		return
	}
	lookupCtx := grpcauthctx.WithUserID(r.Context(), strings.TrimSpace(actor.GetUserId()))
	if strings.EqualFold(strings.TrimSpace(r.FormValue("action")), "verify") {
		verification, err := h.lookupInviteRecipientVerification(lookupCtx, strings.TrimSpace(r.FormValue("recipient_user_id")))
		if err != nil {
			h.renderInviteRecipientLookupError(w, r, err)
			return
		}
		h.renderCampaignInvitesVerificationPage(w, r, campaignID, strings.TrimSpace(actor.GetUserId()), inviteActor.canManageInvites, verification)
		return
	}
	recipientUserID, err := h.resolveInviteRecipientUserID(lookupCtx, strings.TrimSpace(r.FormValue("recipient_user_id")))
	if err != nil {
		h.renderInviteRecipientLookupError(w, r, err)
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), inviteActor.participantID)
	_, err = h.inviteClient.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   targetParticipantID,
		RecipientUserId: recipientUserID,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invite action unavailable", "failed to create invite")
		return
	}

	http.Redirect(w, r, routepath.CampaignInvites(campaignID), http.StatusFound)
}

func (h *handler) handleAppCampaignInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignInviteRevoke removes an invite resource to terminate a
	// pending membership path for the campaign.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	actor, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	inviteActor := h.campaignInviteActorFromParticipant(actor)
	if inviteActor == nil || !inviteActor.canManageInvites {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for invite action")
		return
	}
	if h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invite action unavailable", "campaign invite service is not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "failed to parse invite revoke form")
		return
	}
	inviteID := strings.TrimSpace(r.FormValue("invite_id"))
	if inviteID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invite action unavailable", "invite id is required")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), inviteActor.participantID)
	_, err := h.inviteClient.RevokeInvite(ctx, &statev1.RevokeInviteRequest{
		InviteId: inviteID,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invite action unavailable", "failed to revoke invite")
		return
	}

	http.Redirect(w, r, routepath.CampaignInvites(campaignID), http.StatusFound)
}
