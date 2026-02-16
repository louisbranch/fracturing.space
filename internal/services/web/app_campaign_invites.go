package web

import (
	"context"
	"errors"
	"html"
	"io"
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

func (h *handler) handleAppCampaignInvites(w http.ResponseWriter, r *http.Request, campaignID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}

	actor, err := h.campaignInviteActor(r.Context(), campaignID, sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Invites unavailable", "failed to resolve campaign participant")
		return
	}
	if actor == nil {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "participant access required")
		return
	}
	if !actor.canManageInvites {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "manager or owner access required for invite access")
		return
	}
	if h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invites unavailable", "campaign invite service is not configured")
		return
	}

	ctx := grpcauthctx.WithParticipantID(r.Context(), actor.participantID)
	resp, err := h.inviteClient.ListInvites(ctx, &statev1.ListInvitesRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invites unavailable", "failed to list campaign invites")
		return
	}

	renderAppCampaignInvitesPage(w, campaignID, resp.GetInvites(), actor.canManageInvites)
}

func (h *handler) handleAppCampaignInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	actor, err := h.campaignInviteActor(r.Context(), campaignID, sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Invite action unavailable", "failed to resolve campaign participant")
		return
	}
	if actor == nil {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "participant access required")
		return
	}
	if !actor.canManageInvites {
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

	ctx := grpcauthctx.WithParticipantID(r.Context(), actor.participantID)
	_, err = h.inviteClient.CreateInvite(ctx, &statev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   targetParticipantID,
		RecipientUserId: recipientUserID,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invite action unavailable", "failed to create invite")
		return
	}

	http.Redirect(w, r, "/app/campaigns/"+campaignID+"/invites", http.StatusFound)
}

func (h *handler) handleAppCampaignInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, "/auth/login", http.StatusFound)
		return
	}
	actor, err := h.campaignInviteActor(r.Context(), campaignID, sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Invite action unavailable", "failed to resolve campaign participant")
		return
	}
	if actor == nil {
		h.renderErrorPage(w, r, http.StatusForbidden, "Access denied", "participant access required")
		return
	}
	if !actor.canManageInvites {
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

	ctx := grpcauthctx.WithParticipantID(r.Context(), actor.participantID)
	_, err = h.inviteClient.RevokeInvite(ctx, &statev1.RevokeInviteRequest{
		InviteId: inviteID,
	})
	if err != nil {
		h.renderErrorPage(w, r, grpcErrorHTTPStatus(err, http.StatusBadGateway), "Invite action unavailable", "failed to revoke invite")
		return
	}

	http.Redirect(w, r, "/app/campaigns/"+campaignID+"/invites", http.StatusFound)
}

type campaignInviteActor struct {
	participantID    string
	canManageInvites bool
}

func (h *handler) campaignInviteActor(ctx context.Context, campaignID string, accessToken string) (*campaignInviteActor, error) {
	participant, err := h.campaignParticipant(ctx, campaignID, accessToken)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, nil
	}
	participantID := strings.TrimSpace(participant.GetId())
	if participantID == "" {
		return nil, nil
	}
	return &campaignInviteActor{
		participantID:    participantID,
		canManageInvites: canManageCampaignInvites(participant.GetCampaignAccess()),
	}, nil
}

func (h *handler) campaignParticipant(ctx context.Context, campaignID string, accessToken string) (*statev1.Participant, error) {
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

func renderAppCampaignInvitesPage(w http.ResponseWriter, campaignID string, invites []*statev1.Invite, canManageInvites bool) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	escapedCampaignID := html.EscapeString(campaignID)
	_, _ = io.WriteString(w, "<!doctype html><html><head><title>Campaign Invites</title></head><body><h1>Campaign Invites</h1>")
	if canManageInvites {
		_, _ = io.WriteString(w, "<form method=\"post\" action=\"/app/campaigns/"+escapedCampaignID+"/invites/create\"><input type=\"text\" name=\"participant_id\" placeholder=\"participant id\" required><input type=\"text\" name=\"recipient_user_id\" placeholder=\"recipient user id\"><button type=\"submit\">Create Invite</button></form>")
	}
	_, _ = io.WriteString(w, "<ul>")
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
		_, _ = io.WriteString(w, "<li>"+html.EscapeString(displayInviteID+" - "+recipient))
		if canManageInvites && inviteID != "" {
			_, _ = io.WriteString(w, "<form method=\"post\" action=\"/app/campaigns/"+escapedCampaignID+"/invites/revoke\"><input type=\"hidden\" name=\"invite_id\" value=\""+html.EscapeString(inviteID)+"\"><button type=\"submit\">Revoke</button></form>")
		}
		_, _ = io.WriteString(w, "</li>")
	}
	_, _ = io.WriteString(w, "</ul></body></html>")
}
