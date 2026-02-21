package web

import (
	"context"
	"errors"
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) handleAppCampaignInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignInvites lists invites for a campaign after an actor-level
	// membership lookup, ensuring only authorized members view sensitive flows.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	actor, ok := h.requireCampaignActor(w, r, campaignID)
	if !ok {
		return
	}
	inviteActor := h.campaignInviteActorFromParticipant(actor)
	if inviteActor == nil || !inviteActor.canManageInvites {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for invite access")
		return
	}
	if h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invites unavailable", "campaign invite service is not configured")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), inviteActor.participantID)
	resp, err := h.inviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invites unavailable", "failed to list campaign invites")
		return
	}

	renderAppCampaignInvitesPageWithAppName(w, r, h.resolvedAppName(), campaignID, resp.GetInvites(), inviteActor.canManageInvites)
}

func (h *handler) handleAppCampaignInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignInviteCreate creates a player invitation and binds it to
	// the selected target participant.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
	recipientUserID := strings.TrimSpace(r.FormValue("recipient_user_id"))

	ctx := grpcauthctx.WithParticipantID(r.Context(), inviteActor.participantID)
	_, err := h.inviteClient.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   targetParticipantID,
		RecipientUserId: recipientUserID,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invite action unavailable", "failed to create invite")
		return
	}

	http.Redirect(w, r, "/campaigns/"+campaignID+"/invites", http.StatusFound)
}

func (h *handler) handleAppCampaignInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string) {
	// handleAppCampaignInviteRevoke removes an invite resource to terminate a
	// pending membership path for the campaign.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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

	http.Redirect(w, r, "/campaigns/"+campaignID+"/invites", http.StatusFound)
}

type campaignInviteActor struct {
	participantID    string
	canManageInvites bool
}

func (h *handler) campaignInviteActorFromParticipant(participant *statev1.Participant) *campaignInviteActor {
	if participant == nil {
		return nil
	}
	participantID := strings.TrimSpace(participant.GetId())
	if participantID == "" {
		return nil
	}
	return &campaignInviteActor{
		participantID:    participantID,
		canManageInvites: canManageCampaignInvites(participant.GetCampaignAccess()),
	}
}

func (h *handler) campaignParticipant(ctx context.Context, campaignID string, accessToken string) (*statev1.Participant, error) {
	// campaignParticipant maps an access token to the participant record in the
	// campaign, with pagination across participant pages if needed.
	if h == nil || h.participantClient == nil {
		return nil, errors.New("participant client is not configured")
	}
	userID, err := h.sessionUserID(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	if userID == "" {
		return nil, nil
	}

	pageToken := ""
	for {
		resp, err := h.participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		if err != nil {
			return nil, err
		}
		for _, participant := range resp.GetParticipants() {
			if participant == nil {
				continue
			}
			if strings.TrimSpace(participant.GetUserId()) == userID {
				return participant, nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}

	return nil, nil
}

func canManageCampaignInvites(access statev1.CampaignAccess) bool {
	return access == statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER || access == statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER
}

func renderAppCampaignInvitesPage(w http.ResponseWriter, r *http.Request, campaignID string, invites []*statev1.Invite, canManageInvites bool) {
	renderAppCampaignInvitesPageWithAppName(w, r, "", campaignID, invites, canManageInvites)
}

func renderAppCampaignInvitesPageWithAppName(w http.ResponseWriter, r *http.Request, appName string, campaignID string, invites []*statev1.Invite, canManageInvites bool) {
	// renderAppCampaignInvitesPage exposes write controls only to managed roles.
	campaignID = strings.TrimSpace(campaignID)
	inviteItems := make([]webtemplates.CampaignInviteItem, 0, len(invites))
	for _, invite := range invites {
		if invite == nil {
			continue
		}
		inviteID := strings.TrimSpace(invite.GetId())
		displayInviteID := inviteID
		if displayInviteID == "" {
			displayInviteID = "unknown-invite"
		}
		recipient := strings.TrimSpace(invite.GetRecipientUserId())
		if recipient == "" {
			recipient = "unknown-recipient"
		}
		inviteItems = append(inviteItems, webtemplates.CampaignInviteItem{
			ID:    inviteID,
			Label: displayInviteID + " - " + recipient,
		})
	}
	writeGameContentType(w)
	if err := webtemplates.CampaignInvitesPage(appName, campaignID, canManageInvites, inviteItems).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render campaign invites page", http.StatusInternalServerError)
	}
}
