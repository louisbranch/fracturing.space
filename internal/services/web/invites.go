package web

import (
	"net/http"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

func (h *handler) handleAppInvites(w http.ResponseWriter, r *http.Request) {
	// handleAppInvites lists pending invites for the authenticated user and
	// keeps user identity as the primary partition for this page.
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, routepath.AuthLogin, http.StatusFound)
		return
	}
	if h.inviteClient == nil {
		h.renderErrorPage(w, r, http.StatusServiceUnavailable, "Invites unavailable", "invite service client is not configured")
		return
	}

	userID, err := h.sessionUserIDForSession(r.Context(), sess)
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

	renderAppInvitesPageWithContext(w, r, h.pageContext(w, r), resp.GetInvites())
}

func (h *handler) handleAppInviteClaim(w http.ResponseWriter, r *http.Request) {
	// handleAppInviteClaim exchanges a join grant and claim request to materialize
	// campaign membership for the authenticated user.
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		localizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}

	sess := sessionFromRequest(r, h.sessions)
	if sess == nil {
		http.Redirect(w, r, routepath.AuthLogin, http.StatusFound)
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

	userID, err := h.sessionUserIDForSession(r.Context(), sess)
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

	http.Redirect(w, r, routepath.Campaign(campaignID), http.StatusFound)
}

func renderAppInvitesPage(w http.ResponseWriter, r *http.Request, invites []*statev1.PendingUserInvite) {
	renderAppInvitesPageWithContext(w, r, webtemplates.PageContext{}, invites)
}

func renderAppInvitesPageWithContext(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, invites []*statev1.PendingUserInvite) {
	// renderAppInvitesPage maps pending user invites into the minimal claimable
	// list the web surface exposes.
	mapped := make([]webtemplates.UserInviteItem, 0, len(invites))
	for _, pending := range invites {
		if pending == nil {
			continue
		}
		campaign := pending.GetCampaign()
		invite := pending.GetInvite()
		participant := pending.GetParticipant()

		campaignName := ""
		campaignID := ""
		inviteCampaignID := ""
		inviteID := ""
		if campaign != nil {
			campaignName = strings.TrimSpace(campaign.GetName())
			campaignID = strings.TrimSpace(campaign.GetId())
		}
		if invite != nil {
			inviteCampaignID = strings.TrimSpace(invite.GetCampaignId())
			inviteID = strings.TrimSpace(invite.GetId())
		}

		unknownParticipantLabel := webtemplates.T(page.Loc, "game.invite_item.unknown_participant")
		participantName := unknownParticipantLabel
		participantID := ""
		if participant != nil {
			participantName = strings.TrimSpace(participant.GetName())
			participantID = strings.TrimSpace(participant.GetId())
		}
		if campaignName == "" {
			campaignName = campaignID
		}
		if campaignName == "" {
			campaignName = inviteCampaignID
		}
		if campaignName == "" {
			campaignName = webtemplates.T(page.Loc, "game.invite_item.unknown_campaign")
		}
		if campaignID == "" {
			campaignID = inviteCampaignID
		}
		if participantName == "" {
			participantName = unknownParticipantLabel
		}
		label := campaignName + " - " + participantName
		mapped = append(mapped, webtemplates.UserInviteItem{
			Label:         label,
			CampaignID:    campaignID,
			InviteID:      inviteID,
			ParticipantID: participantID,
		})
	}
	if err := writePage(w, r, webtemplates.UserInvitesPage(page, mapped), composeHTMXTitleForPage(page, "game.my_invites.title")); err != nil {
		localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_invites_page")
	}
}
