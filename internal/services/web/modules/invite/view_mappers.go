package invite

import (
	"strings"

	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// mapPublicInviteView keeps template-facing copy and links out of the app layer.
func mapPublicInviteView(page inviteapp.InvitePage) PublicInvitePageView {
	view := PublicInvitePageView{
		CampaignName:    strings.TrimSpace(page.Invite.CampaignName),
		ParticipantName: strings.TrimSpace(page.Invite.ParticipantName),
		StatusLabel:     strings.Title(string(page.Invite.Status)),
	}
	if username := strings.TrimSpace(page.Invite.InviterUsername); username != "" {
		view.InviterUsername = username
		view.InviterProfileURL = routepath.UserProfile(username)
	}
	switch page.State {
	case inviteapp.InvitePageStateAnonymous:
		view.Heading = "Campaign invitation"
		view.Body = "Sign in or create an account to view and respond to this invitation."
		view.LoginLabel = "Sign in or create account"
		view.LoginURL = loginRedirectForInvite(page.Invite.InviteID)
	case inviteapp.InvitePageStateClaimable:
		view.Heading = "Claim this seat"
		view.Body = "This invitation is unassigned. You can claim it with your current account."
		view.AcceptLabel = "Accept invitation"
		view.AcceptURL = routepath.PublicInviteAccept(page.Invite.InviteID)
	case inviteapp.InvitePageStateTargeted:
		view.Heading = "Invitation ready"
		view.Body = "This invitation is addressed to you. You can accept or decline it now."
		view.AcceptLabel = "Accept invitation"
		view.AcceptURL = routepath.PublicInviteAccept(page.Invite.InviteID)
		view.DeclineLabel = "Decline invitation"
		view.DeclineURL = routepath.PublicInviteDecline(page.Invite.InviteID)
	case inviteapp.InvitePageStateMismatch:
		view.Heading = "Invitation reserved"
		view.Body = "This invitation is reserved for a different account."
		view.DashboardLabel = "Back to dashboard"
		view.DashboardURL = routepath.AppDashboard
	case inviteapp.InvitePageStateClaimed:
		view.Heading = "Invitation claimed"
		view.Body = "This seat has already been claimed."
		view.DashboardLabel = "Back to dashboard"
		view.DashboardURL = routepath.AppDashboard
	case inviteapp.InvitePageStateDeclined:
		view.Heading = "Invitation declined"
		view.Body = "This invitation has already been declined."
		view.DashboardLabel = "Back to dashboard"
		view.DashboardURL = routepath.AppDashboard
	case inviteapp.InvitePageStateRevoked:
		view.Heading = "Invitation revoked"
		view.Body = "This invitation is no longer available."
		view.DashboardLabel = "Back to dashboard"
		view.DashboardURL = routepath.AppDashboard
	default:
		view.Heading = "Campaign invitation"
		view.Body = "Review this invitation."
	}
	return view
}
