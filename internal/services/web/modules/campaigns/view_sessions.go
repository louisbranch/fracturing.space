package campaigns

import (
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
)

// mapSessionsView converts domain sessions to template view items.
func mapSessionsView(items []campaignapp.CampaignSession) []campaignrender.SessionView {
	result := make([]campaignrender.SessionView, 0, len(items))
	for _, session := range items {
		result = append(result, campaignrender.SessionView{
			ID:        session.ID,
			Name:      session.Name,
			Status:    session.Status,
			StartedAt: session.StartedAt,
			UpdatedAt: session.UpdatedAt,
			EndedAt:   session.EndedAt,
		})
	}
	return result
}

// mapSessionReadinessView converts domain readiness state to template view state.
func mapSessionReadinessView(readiness campaignapp.CampaignSessionReadiness) campaignrender.SessionReadinessView {
	result := campaignrender.SessionReadinessView{
		Ready:    readiness.Ready,
		Blockers: make([]campaignrender.SessionReadinessBlockerView, 0, len(readiness.Blockers)),
	}
	for _, blocker := range readiness.Blockers {
		result.Blockers = append(result.Blockers, campaignrender.SessionReadinessBlockerView{
			Code:    blocker.Code,
			Message: blocker.Message,
		})
	}
	return result
}

// mapInvitesView converts domain invites to template view items.
func mapInvitesView(items []campaignapp.CampaignInvite) []campaignrender.InviteView {
	result := make([]campaignrender.InviteView, 0, len(items))
	for _, invite := range items {
		result = append(result, campaignrender.InviteView{
			ID:              invite.ID,
			ParticipantID:   invite.ParticipantID,
			RecipientUserID: invite.RecipientUserID,
			Status:          invite.Status,
		})
	}
	return result
}

// campaignInviteIsPending reports whether an invite should still reserve a seat option.
func campaignInviteIsPending(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "invite_status_pending":
		return true
	default:
		return false
	}
}

// campaignInviteSeatController normalizes controller values used in seat-option display.
func campaignInviteSeatController(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "human", "controller_human":
		return "human"
	case "ai", "controller_ai":
		return "ai"
	default:
		return ""
	}
}
