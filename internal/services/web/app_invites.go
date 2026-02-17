package web

import (
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
)

func (h *handler) handleAppInvites(w http.ResponseWriter, r *http.Request) {
	// handleAppInvites lists pending invites for the authenticated user and
	// keeps user identity as the primary partition for this page.
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
	if h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invites unavailable", "invite service client is not configured")
		return
	}

	userID, err := h.sessionUserID(r.Context(), sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Invites unavailable", "failed to resolve current user")
		return
	}
	if userID == "" {
		h.renderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return
	}

	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	resp, err := h.inviteClient.ListPendingInvitesForUser(ctx, &statev1.ListPendingInvitesForUserRequest{
		PageSize: 10,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Invites unavailable", "failed to list pending invites")
		return
	}

	renderAppInvitesPage(w, resp.GetInvites())
}

func (h *handler) handleAppInviteClaim(w http.ResponseWriter, r *http.Request) {
	// handleAppInviteClaim exchanges a join grant and claim request to materialize
	// campaign membership for the authenticated user.
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
	if h.authClient == nil || h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invite claim unavailable", "invite claim dependencies are not configured")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invalid claim request", "failed to parse claim form")
		return
	}
	campaignID := strings.TrimSpace(r.FormValue("campaign_id"))
	inviteID := strings.TrimSpace(r.FormValue("invite_id"))
	participantID := strings.TrimSpace(r.FormValue("participant_id"))
	if campaignID == "" || inviteID == "" || participantID == "" {
		h.renderErrorPage(w, r, http.StatusBadRequest, "Invalid claim request", "campaign, invite, and participant ids are required")
		return
	}

	userID, err := h.sessionUserID(r.Context(), sess.accessToken)
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Invite claim unavailable", "failed to resolve current user")
		return
	}
	if userID == "" {
		h.renderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return
	}

	grantResp, err := h.authClient.IssueJoinGrant(r.Context(), &authv1.IssueJoinGrantRequest{
		UserId:        userID,
		CampaignId:    campaignID,
		InviteId:      inviteID,
		ParticipantId: participantID,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Invite claim unavailable", "failed to issue join grant")
		return
	}
	joinGrant := strings.TrimSpace(grantResp.GetJoinGrant())
	if joinGrant == "" {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Invite claim unavailable", "join grant was empty")
		return
	}

	ctx := grpcauthctx.WithUserID(r.Context(), userID)
	_, err = h.inviteClient.ClaimInvite(ctx, &statev1.ClaimInviteRequest{
		CampaignId: campaignID,
		InviteId:   inviteID,
		JoinGrant:  joinGrant,
	})
	if err != nil {
		h.renderErrorPage(w, r, http.StatusBadGateway, "Invite claim unavailable", "failed to claim invite")
		return
	}

	http.Redirect(w, r, "/app/campaigns/"+url.PathEscape(campaignID), http.StatusFound)
}

func renderAppInvitesPage(w http.ResponseWriter, invites []*statev1.PendingUserInvite) {
	// renderAppInvitesPage maps pending user invites into the minimal claimable
	// list the web surface exposes.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, "<!doctype html><html><head><title>My Invites</title></head><body><h1>My Invites</h1><ul>")
	for _, pending := range invites {
		if pending == nil {
			continue
		}
		campaignName := strings.TrimSpace(pending.GetCampaign().GetName())
		campaignID := strings.TrimSpace(pending.GetCampaign().GetId())
		if campaignName == "" {
			campaignName = strings.TrimSpace(pending.GetInvite().GetCampaignId())
		}
		if campaignID == "" {
			campaignID = strings.TrimSpace(pending.GetInvite().GetCampaignId())
		}
		if campaignName == "" {
			campaignName = "Unknown campaign"
		}
		participantName := strings.TrimSpace(pending.GetParticipant().GetName())
		participantID := strings.TrimSpace(pending.GetParticipant().GetId())
		inviteID := strings.TrimSpace(pending.GetInvite().GetId())
		if participantName == "" {
			participantName = "Unknown participant"
		}
		label := campaignName + " - " + participantName
		_, _ = io.WriteString(w, "<li>"+html.EscapeString(label))
		if campaignID != "" && inviteID != "" && participantID != "" {
			_, _ = io.WriteString(w, "<form method=\"post\" action=\"/app/invites/claim\"><input type=\"hidden\" name=\"campaign_id\" value=\""+html.EscapeString(campaignID)+"\"><input type=\"hidden\" name=\"invite_id\" value=\""+html.EscapeString(inviteID)+"\"><input type=\"hidden\" name=\"participant_id\" value=\""+html.EscapeString(participantID)+"\"><button type=\"submit\">Claim</button></form>")
		}
		_, _ = io.WriteString(w, "</li>")
	}
	_, _ = io.WriteString(w, "</ul></body></html>")
}
