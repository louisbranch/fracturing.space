package dashboard

import (
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// mapDashboardTemplateView maps values across transport and domain boundaries.
func mapDashboardTemplateView(view dashboardapp.DashboardView) webtemplates.DashboardPageView {
	health := make([]webtemplates.DashboardServiceHealthEntry, len(view.ServiceHealth))
	for i, e := range view.ServiceHealth {
		health[i] = webtemplates.DashboardServiceHealthEntry{Label: e.Label, Available: e.Available}
	}
	activeSessions := make([]webtemplates.DashboardActiveSessionEntry, len(view.ActiveSessions))
	for i, session := range view.ActiveSessions {
		activeSessions[i] = webtemplates.DashboardActiveSessionEntry{
			CampaignID:   session.CampaignID,
			CampaignName: session.CampaignName,
			SessionID:    session.SessionID,
			SessionName:  session.SessionName,
		}
	}
	pendingInvites := make([]webtemplates.DashboardPendingInviteEntry, len(view.PendingInvites))
	for i, invite := range view.PendingInvites {
		pendingInvites[i] = webtemplates.DashboardPendingInviteEntry{
			InviteID:        invite.InviteID,
			CampaignName:    invite.CampaignName,
			ParticipantName: invite.ParticipantName,
		}
	}
	startNudges := make([]webtemplates.DashboardCampaignStartNudgeEntry, len(view.CampaignStartNudges))
	for i, nudge := range view.CampaignStartNudges {
		startNudges[i] = webtemplates.DashboardCampaignStartNudgeEntry{
			CampaignID:          nudge.CampaignID,
			CampaignName:        nudge.CampaignName,
			Message:             nudge.BlockerMessage,
			ActionKind:          string(nudge.ActionKind),
			TargetParticipantID: nudge.TargetParticipantID,
			TargetCharacterID:   nudge.TargetCharacterID,
		}
	}

	statusNotice := webtemplates.DashboardStatusNotice{}
	switch view.DataStatus {
	case dashboardapp.DashboardDataStatusDegraded:
		statusNotice = webtemplates.DashboardStatusNotice{Visible: true, Degraded: true}
	case dashboardapp.DashboardDataStatusUnavailable:
		statusNotice = webtemplates.DashboardStatusNotice{Visible: true}
	}
	return webtemplates.DashboardPageView{
		StatusNotice:   statusNotice,
		ProfilePending: webtemplates.DashboardProfilePendingBlock{Visible: view.ShowPendingProfileBlock},
		PendingInvites: webtemplates.DashboardPendingInvitesBlock{
			Visible: len(pendingInvites) > 0,
			Invites: pendingInvites,
		},
		CampaignStartNudges: webtemplates.DashboardCampaignStartNudgesBlock{
			Visible: len(startNudges) > 0,
			HasMore: view.CampaignStartNudgesMore,
			Nudges:  startNudges,
		},
		Adventure: webtemplates.DashboardAdventureBlock{Visible: view.ShowAdventureBlock},
		ActiveSessions: webtemplates.DashboardActiveSessionsBlock{
			Visible:  len(activeSessions) > 0,
			Sessions: activeSessions,
		},
		ServiceHealth: health,
	}
}
