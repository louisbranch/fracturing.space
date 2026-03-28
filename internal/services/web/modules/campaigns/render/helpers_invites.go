package render

import (
	"strings"

	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// campaignInviteStatusLabel keeps invite tables and detail pages on the same status copy.
func campaignInviteStatusLabel(loc webtemplates.Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return webtemplates.T(loc, "game.campaign_invites.value_unspecified")
	case "pending":
		return webtemplates.T(loc, "game.campaign_invites.value_pending")
	case "claimed":
		return webtemplates.T(loc, "game.campaign_invites.value_claimed")
	case "declined":
		return webtemplates.T(loc, "game.campaign_invites.value_declined")
	case "revoked":
		return webtemplates.T(loc, "game.campaign_invites.value_revoked")
	default:
		return raw
	}
}

// campaignInviteCanRevoke limits revoke affordances to pending invites.
func campaignInviteCanRevoke(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "pending")
}

// campaignInviteCreateReady reports whether the invite-create form can submit.
func campaignInviteCreateReady(view InviteCreatePageView) bool {
	return !campaignActionsLocked(view.ActionsLocked) && len(view.InviteSeatOptions) > 0
}
