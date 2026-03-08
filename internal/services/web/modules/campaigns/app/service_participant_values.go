package app

import "strings"

const (
	participantRoleGMValue     = "gm"
	participantRolePlayerValue = "player"
	participantAccessMember    = "member"
	participantAccessManager   = "manager"
	participantAccessOwner     = "owner"
	participantControllerAI    = "ai"
)

var participantAccessValues = []string{participantAccessMember, participantAccessManager, participantAccessOwner}

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
		return participantAccessMember
	case "manager", "campaign_access_manager":
		return participantAccessManager
	case "owner", "campaign_access_owner":
		return participantAccessOwner
	default:
		return ""
	}
}

// participantControllerCanonical maps transport/view controller labels to canonical values.
func participantControllerCanonical(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ai", "controller_ai":
		return participantControllerAI
	case "human", "controller_human":
		return "human"
	case "unassigned", "controller_unassigned":
		return "unassigned"
	default:
		return ""
	}
}
