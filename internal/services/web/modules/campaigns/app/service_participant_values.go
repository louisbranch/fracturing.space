package app

import "strings"

const (
	participantRoleGMValue     = "gm"
	participantRolePlayerValue = "player"
)

var participantAccessValues = []string{"member", "manager", "owner"}

// participantRoleCanonical maps transport/view role labels to canonical values.
func participantRoleCanonical(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "gm", "participant_role_gm", "role_gm":
		return participantRoleGMValue, true
	case "player", "participant_role_player", "role_player":
		return participantRolePlayerValue, true
	default:
		return "", false
	}
}

// participantAccessCanonical maps transport/view access labels to canonical values.
func participantAccessCanonical(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "member", "campaign_access_member":
		return "member"
	case "manager", "campaign_access_manager":
		return "manager"
	case "owner", "campaign_access_owner":
		return "owner"
	default:
		return ""
	}
}
