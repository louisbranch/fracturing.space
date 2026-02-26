package invites

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/web/support"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

type AppInvitesHandlers struct {
	Authenticate         func(*http.Request) bool
	RedirectToLogin      func(http.ResponseWriter, *http.Request)
	HasInviteClient      func() bool
	HasAuthClient        func() bool
	ResolveProfileUserID func(context.Context) (string, error)
	ListPendingInvites   func(context.Context, *statev1.ListPendingInvitesForUserRequest) (*statev1.ListPendingInvitesForUserResponse, error)
	IssueJoinGrant       func(context.Context, *authv1.IssueJoinGrantRequest) (*authv1.IssueJoinGrantResponse, error)
	ClaimInvite          func(context.Context, *statev1.ClaimInviteRequest) error
	RenderErrorPage      func(http.ResponseWriter, *http.Request, int, string, string)
	PageContext          func(*http.Request) webtemplates.PageContext
}

// HandleAppInvites handles /app/invites.
func HandleAppInvites(h AppInvitesHandlers, w http.ResponseWriter, r *http.Request) {
	if h.Authenticate == nil ||
		h.RedirectToLogin == nil ||
		h.ResolveProfileUserID == nil ||
		h.RenderErrorPage == nil ||
		h.PageContext == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		support.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if !h.Authenticate(r) {
		h.RedirectToLogin(w, r)
		return
	}
	if !h.HasInviteClient() {
		h.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Invites unavailable", "invite service client is not configured")
		return
	}
	userID, err := h.ResolveProfileUserID(r.Context())
	if err != nil {
		h.RenderErrorPage(w, r, http.StatusBadGateway, "Invites unavailable", "failed to resolve current user")
		return
	}
	if strings.TrimSpace(userID) == "" {
		h.RenderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return
	}

	userCtx := grpcauthctx.WithUserID(r.Context(), userID)
	resp, err := h.ListPendingInvites(userCtx, &statev1.ListPendingInvitesForUserRequest{
		PageSize: 10,
	})
	if err != nil {
		h.RenderErrorPage(w, r, http.StatusBadGateway, "Invites unavailable", "failed to list pending invites")
		return
	}
	RenderAppInvitesPage(w, r, h.PageContext(r), resp.GetInvites())
}

// HandleAppInviteClaim handles /app/invites/claim.
func HandleAppInviteClaim(h AppInvitesHandlers, w http.ResponseWriter, r *http.Request) {
	if h.Authenticate == nil ||
		h.RedirectToLogin == nil ||
		h.ResolveProfileUserID == nil ||
		h.RenderErrorPage == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		support.LocalizeHTTPError(w, r, http.StatusMethodNotAllowed, "error.http.method_not_allowed")
		return
	}
	if !h.Authenticate(r) {
		h.RedirectToLogin(w, r)
		return
	}
	if h.HasAuthClient == nil || h.HasInviteClient == nil || !h.HasAuthClient() || !h.HasInviteClient() {
		h.RenderErrorPage(w, r, http.StatusServiceUnavailable, "Invite claim unavailable", "invite claim dependencies are not configured")
		return
	}
	if h.IssueJoinGrant == nil || h.ClaimInvite == nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		h.RenderErrorPage(w, r, http.StatusBadRequest, "Invalid claim request", "failed to parse claim form")
		return
	}
	campaignID := strings.TrimSpace(r.FormValue("campaign_id"))
	inviteID := strings.TrimSpace(r.FormValue("invite_id"))
	participantID := strings.TrimSpace(r.FormValue("participant_id"))
	if campaignID == "" || inviteID == "" || participantID == "" {
		h.RenderErrorPage(w, r, http.StatusBadRequest, "Invalid claim request", "campaign, invite, and participant ids are required")
		return
	}

	userID, err := h.ResolveProfileUserID(r.Context())
	if err != nil {
		h.RenderErrorPage(w, r, http.StatusBadGateway, "Invite claim unavailable", "failed to resolve current user")
		return
	}
	if strings.TrimSpace(userID) == "" {
		h.RenderErrorPage(w, r, http.StatusUnauthorized, "Authentication required", "no user identity was resolved for this session")
		return
	}
	userCtx := grpcauthctx.WithUserID(r.Context(), userID)
	grantResp, err := h.IssueJoinGrant(userCtx, &authv1.IssueJoinGrantRequest{
		UserId:        userID,
		CampaignId:    campaignID,
		InviteId:      inviteID,
		ParticipantId: participantID,
	})
	if err != nil {
		h.RenderErrorPage(w, r, http.StatusBadGateway, "Invite claim unavailable", "failed to issue join grant")
		return
	}
	joinGrant := strings.TrimSpace(grantResp.GetJoinGrant())
	if joinGrant == "" {
		h.RenderErrorPage(w, r, http.StatusBadGateway, "Invite claim unavailable", "join grant was empty")
		return
	}
	if err := h.ClaimInvite(userCtx, &statev1.ClaimInviteRequest{
		CampaignId: campaignID,
		InviteId:   inviteID,
		JoinGrant:  joinGrant,
	}); err != nil {
		h.RenderErrorPage(w, r, http.StatusBadGateway, "Invite claim unavailable", "failed to claim invite")
		return
	}
	http.Redirect(w, r, routepath.Campaign(campaignID), http.StatusFound)
}

// RenderAppInvitesPage renders the pending invites list for the current user.
func RenderAppInvitesPage(w http.ResponseWriter, r *http.Request, page webtemplates.PageContext, invites []*statev1.PendingUserInvite) {
	if err := support.WritePage(
		w,
		r,
		RenderAppInvitesPageWithContext(page, invites),
		support.ComposeHTMXTitleForPage(page, "game.my_invites.title"),
	); err != nil {
		support.LocalizeHTTPError(w, r, http.StatusInternalServerError, "error.http.failed_to_render_invites_page")
	}
}

// RenderAppInvitesPageWithContext builds invite list component without requiring handlers.
func RenderAppInvitesPageWithContext(page webtemplates.PageContext, invites []*statev1.PendingUserInvite) templ.Component {
	return webtemplates.UserInvitesPage(page, mapPendingInvites(invites, page))
}

func mapPendingInvites(invites []*statev1.PendingUserInvite, page webtemplates.PageContext) []webtemplates.UserInviteItem {
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
	return mapped
}
