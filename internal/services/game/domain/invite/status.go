package invite

import "strings"

// NormalizeStatusLabel canonicalizes invite status labels for stable payload hashes.
//
// Invite lifecycle is shared across API, projections, and storage, so status
// normalization prevents spelling/casing drift from creating separate replay paths.
func NormalizeStatusLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "PENDING", "INVITE_STATUS_PENDING":
		return statusPending, true
	case "CLAIMED", "INVITE_STATUS_CLAIMED":
		return statusClaimed, true
	case "REVOKED", "INVITE_STATUS_REVOKED":
		return statusRevoked, true
	case "DECLINED", "INVITE_STATUS_DECLINED":
		return statusDeclined, true
	default:
		return "", false
	}
}
