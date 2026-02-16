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
func NormalizeGateType(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("gate type is required")
	}
	return strings.ToLower(trimmed), nil
}

// NormalizeGateReason trims a gate reason string.
func NormalizeGateReason(value string) string {
	return strings.TrimSpace(value)
}

// NormalizeSpotlightType validates and normalizes a spotlight type value.
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
