package gate

import (
	"fmt"
	"strings"
)

// GateStatus identifies the session gate lifecycle label.
type GateStatus string

const (
	GateStatusOpen      GateStatus = "open"
	GateStatusResolved  GateStatus = "resolved"
	GateStatusAbandoned GateStatus = "abandoned"
)

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
