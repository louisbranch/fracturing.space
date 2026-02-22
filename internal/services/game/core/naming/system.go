// Package naming provides shared system namespace normalization used by
// command, event, and engine registries to enforce consistent naming across
// pluggable game-system modules.
package naming

import (
	"fmt"
	"strings"
)

// NormalizeSystemNamespace converts a raw system identifier (e.g.
// "GAME_SYSTEM_ALPHA", "my-system") into the canonical lowercase
// underscore-separated namespace used in type prefixes ("alpha",
// "my_system").
func NormalizeSystemNamespace(systemID string) string {
	trimmed := strings.TrimSpace(systemID)
	if trimmed == "" {
		return ""
	}
	const legacyPrefix = "GAME_SYSTEM_"
	if len(trimmed) > len(legacyPrefix) && strings.EqualFold(trimmed[:len(legacyPrefix)], legacyPrefix) {
		trimmed = trimmed[len(legacyPrefix):]
	}
	normalized := strings.ToLower(trimmed)
	var b strings.Builder
	b.Grow(len(normalized))
	lastUnderscore := false
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

// ValidateSystemNamespace checks that a system-prefixed type name's namespace
// matches the normalized systemID. Non-system types and empty systemIDs are
// silently accepted so callers only need one call site for both ownership cases.
func ValidateSystemNamespace(typeName, systemID string) error {
	if systemID == "" {
		return nil
	}
	expectedNS, ok := NamespaceFromType(typeName)
	if !ok {
		return nil
	}
	if NormalizeSystemNamespace(systemID) != expectedNS {
		return fmt.Errorf("system id %s does not match type namespace %s in %s",
			systemID, expectedNS, typeName)
	}
	return nil
}

// NamespaceFromType extracts the system namespace from a "sys.{namespace}.…"
// type name. It returns ("", false) for non-system types.
func NamespaceFromType(typeName string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(typeName), ".")
	if len(parts) < 3 || parts[0] != "sys" {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}
