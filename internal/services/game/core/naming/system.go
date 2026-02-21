// Package naming provides shared system namespace normalization used by
// command, event, and engine registries to enforce consistent naming across
// pluggable game-system modules.
package naming

import "strings"

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

// NamespaceFromType extracts the system namespace from a "sys.{namespace}.…"
// type name. It returns ("", false) for non-system types.
func NamespaceFromType(typeName string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(typeName), ".")
	if len(parts) < 3 || parts[0] != "sys" {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}
