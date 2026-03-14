package dashboard

import (
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
)

// mapDashboardTemplateView maps values across transport and domain boundaries.
func mapDashboardTemplateView(view dashboardapp.DashboardView) DashboardPageView {
	health := make([]DashboardServiceHealthEntry, len(view.ServiceHealth))
	for i, e := range view.ServiceHealth {
		health[i] = DashboardServiceHealthEntry{Label: e.Label, Available: e.Available}
	}
	activeSessions := make([]DashboardActiveSessionEntry, len(view.ActiveSessions))
	for i, session := range view.ActiveSessions {
		activeSessions[i] = DashboardActiveSessionEntry{
			CampaignID:   session.CampaignID,
			CampaignName: session.CampaignName,
			SessionID:    session.SessionID,
			SessionName:  session.SessionName,
		}
	}
	pendingInvites := make([]DashboardPendingInviteEntry, len(view.PendingInvites))
	for i, invite := range view.PendingInvites {
		pendingInvites[i] = DashboardPendingInviteEntry{
			InviteID:        invite.InviteID,
			CampaignName:    invite.CampaignName,
			ParticipantName: invite.ParticipantName,
		}
	}
	startNudges := make([]DashboardCampaignStartNudgeEntry, len(view.CampaignStartNudges))
	for i, nudge := range view.CampaignStartNudges {
		startNudges[i] = DashboardCampaignStartNudgeEntry{
			CampaignID:          nudge.CampaignID,
			CampaignName:        nudge.CampaignName,
			Message:             nudge.BlockerMessage,
			ActionKind:          string(nudge.ActionKind),
			TargetParticipantID: nudge.TargetParticipantID,
			TargetCharacterID:   nudge.TargetCharacterID,
		}
	}

	statusNotice := DashboardStatusNotice{}
	switch view.DataStatus {
	case dashboardapp.DashboardDataStatusDegraded:
		statusNotice = DashboardStatusNotice{Visible: true, Degraded: true}
	case dashboardapp.DashboardDataStatusUnavailable:
		statusNotice = DashboardStatusNotice{Visible: true}
	}
	return DashboardPageView{
		StatusNotice:   statusNotice,
		ProfilePending: DashboardProfilePendingBlock{Visible: view.ShowPendingProfileBlock},
		PendingInvites: DashboardPendingInvitesBlock{
			Visible: len(pendingInvites) > 0,
			Invites: pendingInvites,
		},
		CampaignStartNudges: DashboardCampaignStartNudgesBlock{
			Visible: len(startNudges) > 0,
			HasMore: view.CampaignStartNudgesMore,
			Nudges:  startNudges,
		},
		Adventure: DashboardAdventureBlock{Visible: view.ShowAdventureBlock},
		ActiveSessions: DashboardActiveSessionsBlock{
			Visible:  len(activeSessions) > 0,
			Sessions: activeSessions,
		},
		ServiceHealth: health,
	}
}
