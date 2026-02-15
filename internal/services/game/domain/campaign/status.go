package campaign

import "strings"

// Status describes the campaign lifecycle label used by domain decisions.
type Status string

const (
	StatusUnspecified Status = ""
	StatusDraft       Status = "draft"
	StatusActive      Status = "active"
	StatusCompleted   Status = "completed"
	StatusArchived    Status = "archived"
)

// normalizeStatusLabel canonicalizes status labels for stable payload hashes.
func normalizeStatusLabel(value string) (Status, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "DRAFT", "CAMPAIGN_STATUS_DRAFT":
		return StatusDraft, true
	case "ACTIVE", "CAMPAIGN_STATUS_ACTIVE":
		return StatusActive, true
	case "COMPLETED", "CAMPAIGN_STATUS_COMPLETED":
		return StatusCompleted, true
	case "ARCHIVED", "CAMPAIGN_STATUS_ARCHIVED":
		return StatusArchived, true
	default:
		return "", false
	}
}

// isStatusTransitionAllowed enforces valid campaign lifecycle transitions.
func isStatusTransitionAllowed(from, to Status) bool {
	switch from {
	case StatusDraft:
		return to == StatusActive
	case StatusActive:
		return to == StatusCompleted || to == StatusArchived
	case StatusCompleted:
		return to == StatusArchived
	case StatusArchived:
		return to == StatusDraft
	default:
		return false
	}
}

// IsStatusTransitionAllowed reports whether a status transition is permitted.
func IsStatusTransitionAllowed(from, to Status) bool {
	return isStatusTransitionAllowed(from, to)
}
