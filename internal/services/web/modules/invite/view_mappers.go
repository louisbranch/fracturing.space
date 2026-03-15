package invite

import (
	"strings"

	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// mapPublicInviteView keeps template-facing copy and links out of the app layer.
func mapPublicInviteView(page inviteapp.InvitePage, loc Localizer) PublicInvitePageView {
	view := PublicInvitePageView{
		CampaignName:    strings.TrimSpace(page.Invite.CampaignName),
		ParticipantName: strings.TrimSpace(page.Invite.ParticipantName),
		StatusLabel:     inviteStatusLabel(page.Invite.Status, loc),
	}
	if username := strings.TrimSpace(page.Invite.InviterUsername); username != "" {
		view.InviterUsername = username
		view.InviterProfileURL = routepath.UserProfile(username)
	}
	switch page.State {
	case inviteapp.InvitePageStateAnonymous:
		view.Heading = T(loc, "web.invite.state.anonymous.heading")
		view.Body = T(loc, "web.invite.state.anonymous.body")
		view.LoginLabel = T(loc, "web.invite.action.login")
		view.LoginURL = loginRedirectForInvite(page.Invite.InviteID)
	case inviteapp.InvitePageStateClaimable:
		view.Heading = T(loc, "web.invite.state.claimable.heading")
		view.Body = T(loc, "web.invite.state.claimable.body")
		view.AcceptLabel = T(loc, "web.invite.action.accept")
		view.AcceptURL = routepath.PublicInviteAccept(page.Invite.InviteID)
	case inviteapp.InvitePageStateTargeted:
		view.Heading = T(loc, "web.invite.state.targeted.heading")
		view.Body = T(loc, "web.invite.state.targeted.body")
		view.AcceptLabel = T(loc, "web.invite.action.accept")
		view.AcceptURL = routepath.PublicInviteAccept(page.Invite.InviteID)
		view.DeclineLabel = T(loc, "web.invite.action.decline")
		view.DeclineURL = routepath.PublicInviteDecline(page.Invite.InviteID)
	case inviteapp.InvitePageStateMismatch:
		view.Heading = T(loc, "web.invite.state.mismatch.heading")
		view.Body = T(loc, "web.invite.state.mismatch.body")
		view.DashboardLabel = T(loc, "web.invite.action.dashboard")
		view.DashboardURL = routepath.AppDashboard
	case inviteapp.InvitePageStateClaimed:
		view.Heading = T(loc, "web.invite.state.claimed.heading")
		view.Body = T(loc, "web.invite.state.claimed.body")
		view.DashboardLabel = T(loc, "web.invite.action.dashboard")
		view.DashboardURL = routepath.AppDashboard
	case inviteapp.InvitePageStateDeclined:
		view.Heading = T(loc, "web.invite.state.declined.heading")
		view.Body = T(loc, "web.invite.state.declined.body")
		view.DashboardLabel = T(loc, "web.invite.action.dashboard")
		view.DashboardURL = routepath.AppDashboard
	case inviteapp.InvitePageStateRevoked:
		view.Heading = T(loc, "web.invite.state.revoked.heading")
		view.Body = T(loc, "web.invite.state.revoked.body")
		view.DashboardLabel = T(loc, "web.invite.action.dashboard")
		view.DashboardURL = routepath.AppDashboard
	default:
		view.Heading = T(loc, "web.invite.state.default.heading")
		view.Body = T(loc, "web.invite.state.default.body")
	}
	return view
}

// inviteStatusLabel resolves invite lifecycle status copy at the invite
// rendering seam so transport state stays locale-neutral.
func inviteStatusLabel(status inviteapp.InviteStatus, loc Localizer) string {
	switch status {
	case inviteapp.InviteStatusPending:
		return T(loc, "web.invite.status.pending")
	case inviteapp.InviteStatusClaimed:
		return T(loc, "web.invite.status.claimed")
	case inviteapp.InviteStatusDeclined:
		return T(loc, "web.invite.status.declined")
	case inviteapp.InviteStatusRevoked:
		return T(loc, "web.invite.status.revoked")
	default:
		return T(loc, "web.invite.status.unspecified")
	}
}
