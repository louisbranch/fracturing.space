package validator

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
)

// ValidateStatModifierChangePayload validates a stat_modifier.change command payload.
func ValidateStatModifierChangePayload(raw json.RawMessage) error {
	var p payload.StatModifierChangePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if p.ModifiersAfter == nil {
		return errors.New("modifiers_after is required")
	}
	return nil
}

// ValidateStatModifierChangedPayload validates a stat_modifier_changed event payload.
func ValidateStatModifierChangedPayload(raw json.RawMessage) error {
	var p payload.StatModifierChangedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if p.Modifiers == nil {
		return errors.New("modifiers_after is required")
	}
	return nil
}
