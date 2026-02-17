package invite

import "strings"

// normalizeStatusLabel canonicalizes invite status labels for stable payload hashes.
//
// Invite lifecycle is shared across API, projections, and storage, so status
// normalization prevents spelling/casing drift from creating separate replay paths.
func normalizeStatusLabel(value string) (string, bool) {
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
	default:
		return "", false
	}
}
