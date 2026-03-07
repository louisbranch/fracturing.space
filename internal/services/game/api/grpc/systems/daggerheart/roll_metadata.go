package daggerheart

import "strings"

// rollModifierMetadata captures one normalized modifier entry in roll metadata.
type rollModifierMetadata struct {
	Value  int    `json:"value"`
	Source string `json:"source,omitempty"`
}

// rollSystemMetadata captures the typed `system_data` contract for roll-resolved
// payloads used by Daggerheart transport workflows.
type rollSystemMetadata struct {
	CharacterID       string                 `json:"character_id,omitempty"`
	AdversaryID       string                 `json:"adversary_id,omitempty"`
	Trait             string                 `json:"trait,omitempty"`
	RollKind          string                 `json:"roll_kind,omitempty"`
	Outcome           string                 `json:"outcome,omitempty"`
	Flavor            string                 `json:"flavor,omitempty"`
	BreathCountdownID string                 `json:"breath_countdown_id,omitempty"`
	HopeFear          *bool                  `json:"hope_fear,omitempty"`
	Crit              *bool                  `json:"crit,omitempty"`
	CritNegates       *bool                  `json:"crit_negates,omitempty"`
	GMMove            *bool                  `json:"gm_move,omitempty"`
	Underwater        *bool                  `json:"underwater,omitempty"`
	Roll              *int                   `json:"roll,omitempty"`
	Modifier          *int                   `json:"modifier,omitempty"`
	Total             *int                   `json:"total,omitempty"`
	BaseTotal         *int                   `json:"base_total,omitempty"`
	Critical          *bool                  `json:"critical,omitempty"`
	CriticalBonus     *int                   `json:"critical_bonus,omitempty"`
	Advantage         *int                   `json:"advantage,omitempty"`
	Disadvantage      *int                   `json:"disadvantage,omitempty"`
	Modifiers         []rollModifierMetadata `json:"modifiers,omitempty"`
}

// mapValue converts typed roll metadata into the map-form expected by
// action.RollResolvePayload while keeping key names canonical.
func (m rollSystemMetadata) mapValue() map[string]any {
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

	putString(sdKeyCharacterID, m.CharacterID)
	putString(sdKeyAdversaryID, m.AdversaryID)
	putString("trait", m.Trait)
	putString(sdKeyRollKind, m.RollKind)
	putString(sdKeyOutcome, m.Outcome)
	putString("flavor", m.Flavor)
	putString("breath_countdown_id", m.BreathCountdownID)

	putBool(sdKeyHopeFear, m.HopeFear)
	putBool(sdKeyCrit, m.Crit)
	putBool(sdKeyCritNegates, m.CritNegates)
	putBool("gm_move", m.GMMove)
	putBool("underwater", m.Underwater)
	putBool("critical", m.Critical)

	putInt(sdKeyRoll, m.Roll)
	putInt(sdKeyModifier, m.Modifier)
	putInt(sdKeyTotal, m.Total)
	putInt("base_total", m.BaseTotal)
	putInt("critical_bonus", m.CriticalBonus)
	putInt("advantage", m.Advantage)
	putInt("disadvantage", m.Disadvantage)

	if len(m.Modifiers) > 0 {
		data["modifiers"] = m.Modifiers
	}

	return data
}

func boolPtr(value bool) *bool { return &value }

func intPtrValue(value int) *int { return &value }
