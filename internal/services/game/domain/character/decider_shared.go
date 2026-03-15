package character

import "strings"

// normalizeCharacterKindLabel returns a canonical character kind label.
//
// Character kinds flow into character-sheet and system-specific behavior, so this
// normalization prevents mismatched kind values from bifurcating state.
func normalizeCharacterKindLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "PC", "CHARACTER_KIND_PC":
		return "pc", true
	case "NPC", "CHARACTER_KIND_NPC":
		return "npc", true
	default:
		return "", false
	}
}
