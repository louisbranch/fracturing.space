package character

import "strings"

// Kind identifies the character kind label.
type Kind string

const (
	KindUnspecified Kind = ""
	KindPC          Kind = "pc"
	KindNPC         Kind = "npc"
)

// NormalizeKind parses a kind label into a canonical value.
func NormalizeKind(value string) (Kind, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return KindUnspecified, false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "PC", "CHARACTER_KIND_PC":
		return KindPC, true
	case "NPC", "CHARACTER_KIND_NPC":
		return KindNPC, true
	default:
		return KindUnspecified, false
	}
}
