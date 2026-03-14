package campaigns

import (
	"net/http"
	"net/url"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
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
func mapInvitesView(items []campaignapp.CampaignInvite, r *http.Request) []campaignrender.InviteView {
	result := make([]campaignrender.InviteView, 0, len(items))
	for _, invite := range items {
		result = append(result, campaignrender.InviteView{
			ID:                invite.ID,
			ParticipantID:     invite.ParticipantID,
			ParticipantName:   invite.ParticipantName,
			RecipientUsername: invite.RecipientUsername,
			HasRecipient:      invite.HasRecipient,
			PublicURL:         absolutePublicInviteURL(r, invite.ID),
			Status:            invite.Status,
		})
	}
	return result
}

// absolutePublicInviteURL builds a same-origin invite URL so campaign managers
// can copy the public claim link directly from the management card.
func absolutePublicInviteURL(r *http.Request, inviteID string) string {
	path := routepath.PublicInvite(inviteID)
	if r == nil {
		return path
	}

	scheme := "http"
	if requestmeta.IsHTTPS(r) {
		scheme = "https"
	}

	host := strings.TrimSpace(r.Host)
	if host == "" && r.URL != nil {
		host = strings.TrimSpace(r.URL.Host)
	}
	if host == "" {
		return path
	}

	return (&url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}).String()
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
