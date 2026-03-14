package render

import "strings"

// campaignInviteStatusLabel keeps invite tables and detail pages on the same status copy.
func campaignInviteStatusLabel(loc Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return T(loc, "game.campaign_invites.value_unspecified")
	case "pending":
		return T(loc, "game.campaign_invites.value_pending")
	case "claimed":
		return T(loc, "game.campaign_invites.value_claimed")
	case "declined":
		return T(loc, "game.campaign_invites.value_declined")
	case "revoked":
		return T(loc, "game.campaign_invites.value_revoked")
	default:
		return raw
	}
}

// campaignInviteCanRevoke limits revoke affordances to pending invites.
func campaignInviteCanRevoke(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "pending")
}
