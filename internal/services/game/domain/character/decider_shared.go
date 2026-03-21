package character

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

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

// acceptCharacterEvent creates the standard character event envelope for
// accepted commands. Centralizing this constructor keeps character event
// metadata consistent across all character command handlers.
func acceptCharacterEvent(cmd command.Command, now func() time.Time, eventType event.Type, characterID string, payload any) command.Decision {
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, eventType, "character", characterID, payloadJSON, now().UTC())
	return command.Accept(evt)
}
