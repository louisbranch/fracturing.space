package workflowtransport

import "strings"

// MapValue converts typed roll metadata into the map-form expected by
// action.RollResolvePayload while keeping key names canonical.
func (m RollSystemMetadata) MapValue() map[string]any {
	data := make(map[string]any)

	putString := func(key, value string) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			data[key] = trimmed
		}
	}
	putBool := func(key string, value *bool) {
		if value != nil {
			data[key] = *value
		}
	}
	putInt := func(key string, value *int) {
		if value != nil {
			data[key] = *value
		}
	}

	putString(KeyCharacterID, m.CharacterID)
	putString(KeyAdversaryID, m.AdversaryID)
	putString("trait", m.Trait)
	putString(KeyRollKind, m.RollKind)
	putString(KeyOutcome, m.Outcome)
	putString("flavor", m.Flavor)
	putString("breath_countdown_id", m.BreathCountdownID)

	putBool(KeyHopeFear, m.HopeFear)
	putBool(KeyCrit, m.Crit)
	putBool(KeyCritNegates, m.CritNegates)
	putBool("gm_move", m.GMMove)
	putBool("underwater", m.Underwater)
	putBool("critical", m.Critical)

	putInt(KeyRoll, m.Roll)
	putInt(KeyModifier, m.Modifier)
	putInt(KeyTotal, m.Total)
	putInt("base_total", m.BaseTotal)
	putInt("critical_bonus", m.CriticalBonus)
	putInt("advantage", m.Advantage)
	putInt("disadvantage", m.Disadvantage)

	if len(m.Modifiers) > 0 {
		data["modifiers"] = m.Modifiers
	}

	return data
}

// BoolPtr returns a heap-stable bool pointer for workflow metadata.
func BoolPtr(value bool) *bool { return &value }

// IntPtr returns a heap-stable int pointer for workflow metadata.
func IntPtr(value int) *int { return &value }

// BoolValue dereferences a bool pointer, falling back when nil.
func BoolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

// IntValue dereferences an int pointer and reports whether a value was present.
func IntValue(value *int) (int, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}
