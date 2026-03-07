package scene

import (
	"fmt"
	"strings"
)

// SpotlightType identifies who has the spotlight within a scene.
type SpotlightType string

const (
	SpotlightTypeGM        SpotlightType = "gm"
	SpotlightTypeCharacter SpotlightType = "character"
)

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

// NormalizeGateType validates and normalizes a gate type value.
func NormalizeGateType(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("gate type is required")
	}
	return strings.ToLower(trimmed), nil
}
