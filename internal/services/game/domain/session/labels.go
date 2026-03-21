package session

import (
	"fmt"
	"strings"
)

// Status identifies the session lifecycle label.
type Status string

const (
	StatusUnspecified Status = ""
	StatusActive      Status = "active"
	StatusEnded       Status = "ended"
)

// SpotlightType identifies who has the spotlight.
type SpotlightType string

const (
	SpotlightTypeGM        SpotlightType = "gm"
	SpotlightTypeCharacter SpotlightType = "character"
)

// AITurnStatus identifies the authoritative AI GM turn lifecycle for the
// current GM-owned interaction moment.
type AITurnStatus string

const (
	AITurnStatusIdle    AITurnStatus = "idle"
	AITurnStatusQueued  AITurnStatus = "queued"
	AITurnStatusRunning AITurnStatus = "running"
	AITurnStatusFailed  AITurnStatus = "failed"
)

// NormalizeStatus parses a session status label into a canonical value.
//
// Normalization keeps API and test payloads aligned so status comparisons stay
// stable across replay and branching flows.
func NormalizeStatus(value string) (Status, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return StatusUnspecified, false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "ACTIVE", "SESSION_STATUS_ACTIVE":
		return StatusActive, true
	case "ENDED", "SESSION_STATUS_ENDED":
		return StatusEnded, true
	default:
		return StatusUnspecified, false
	}
}

// NormalizeSpotlightType validates and normalizes a spotlight type value.
//
// The spotlight type controls who can consume command outcomes at a given moment.
func NormalizeSpotlightType(value string) (SpotlightType, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "", fmt.Errorf("spotlight type is required")
	}
	switch SpotlightType(trimmed) {
	case SpotlightTypeGM, SpotlightTypeCharacter:
		return SpotlightType(trimmed), nil
	default:
		return "", fmt.Errorf("spotlight type %q is not supported", value)
	}
}

// ValidateSpotlightTarget enforces target requirements based on spotlight type.
//
// This guard protects command handlers from routing spotlight to an invalid target.
func ValidateSpotlightTarget(spotlightType SpotlightType, characterID string) error {
	characterID = strings.TrimSpace(characterID)
	if spotlightType == SpotlightTypeCharacter && characterID == "" {
		return fmt.Errorf("spotlight character id is required")
	}
	if spotlightType == SpotlightTypeGM && characterID != "" {
		return fmt.Errorf("spotlight character id must be empty for gm spotlight")
	}
	return nil
}

// NormalizeAITurnStatus validates and canonicalizes an AI turn status label.
func NormalizeAITurnStatus(value string) (AITurnStatus, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch AITurnStatus(trimmed) {
	case AITurnStatusIdle, AITurnStatusQueued, AITurnStatusRunning, AITurnStatusFailed:
		return AITurnStatus(trimmed), nil
	default:
		return "", fmt.Errorf("ai turn status %q is not supported", value)
	}
}
