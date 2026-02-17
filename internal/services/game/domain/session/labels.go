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

// GateStatus identifies the session gate lifecycle label.
type GateStatus string

const (
	GateStatusOpen      GateStatus = "open"
	GateStatusResolved  GateStatus = "resolved"
	GateStatusAbandoned GateStatus = "abandoned"
)

// SpotlightType identifies who has the spotlight.
type SpotlightType string

const (
	SpotlightTypeGM        SpotlightType = "gm"
	SpotlightTypeCharacter SpotlightType = "character"
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

// NormalizeGateType validates and normalizes a gate type value.
//
// Gate types are intentionally free-form at storage level, but validated here so
// invalid commands cannot open a gate with malformed metadata.
func NormalizeGateType(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("gate type is required")
	}
	return strings.ToLower(trimmed), nil
}

// NormalizeGateReason trims a gate reason string.
//
// Even optional strings are normalized to reduce irrelevant replay diffs.
func NormalizeGateReason(value string) string {
	return strings.TrimSpace(value)
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
