package snapstate

import "strings"

// CharacterUnstoppableState tracks the Guardian's temporary unstoppable state.
type CharacterUnstoppableState struct {
	Active           bool `json:"active"`
	CurrentValue     int  `json:"current_value"`
	DieSides         int  `json:"die_sides"`
	UsedThisLongRest bool `json:"used_this_long_rest"`
}

// CharacterDamageDie stores one resolved die spec in class-owned runtime state.
type CharacterDamageDie struct {
	Count int `json:"count"`
	Sides int `json:"sides"`
}

// CharacterActiveBeastformState stores the resolved runtime beastform snapshot
// used by attack and damage flows.
type CharacterActiveBeastformState struct {
	BeastformID            string               `json:"beastform_id,omitempty"`
	BaseTrait              string               `json:"base_trait,omitempty"`
	AttackTrait            string               `json:"attack_trait,omitempty"`
	TraitBonus             int                  `json:"trait_bonus,omitempty"`
	EvasionBonus           int                  `json:"evasion_bonus,omitempty"`
	AttackRange            string               `json:"attack_range,omitempty"`
	DamageDice             []CharacterDamageDie `json:"damage_dice,omitempty"`
	DamageBonus            int                  `json:"damage_bonus,omitempty"`
	DamageType             string               `json:"damage_type,omitempty"`
	EvolutionTraitOverride string               `json:"evolution_trait_override,omitempty"`
	DropOnAnyHPMark        bool                 `json:"drop_on_any_hp_mark,omitempty"`
}

// CharacterClassState stores mutable class-owned runtime state that cannot
// live in the static Daggerheart profile.
type CharacterClassState struct {
	AttackBonusUntilRest            int                            `json:"attack_bonus_until_rest,omitempty"`
	EvasionBonusUntilHitOrRest      int                            `json:"evasion_bonus_until_hit_or_rest,omitempty"`
	DifficultyPenaltyUntilRest      int                            `json:"difficulty_penalty_until_rest,omitempty"`
	FocusTargetID                   string                         `json:"focus_target_id,omitempty"`
	ActiveBeastform                 *CharacterActiveBeastformState `json:"active_beastform,omitempty"`
	StrangePatternsNumber           int                            `json:"strange_patterns_number,omitempty"`
	RallyDice                       []int                          `json:"rally_dice,omitempty"`
	PrayerDice                      []int                          `json:"prayer_dice,omitempty"`
	Unstoppable                     CharacterUnstoppableState      `json:"unstoppable,omitempty"`
	ChannelRawPowerUsedThisLongRest bool                           `json:"channel_raw_power_used_this_long_rest,omitempty"`
}

// Normalized clamps empty/invalid class-state values so every write path
// persists the same runtime shape.
func (s CharacterClassState) Normalized() CharacterClassState {
	normalized := s
	if normalized.AttackBonusUntilRest < 0 {
		normalized.AttackBonusUntilRest = 0
	}
	if normalized.EvasionBonusUntilHitOrRest < 0 {
		normalized.EvasionBonusUntilHitOrRest = 0
	}
	if normalized.DifficultyPenaltyUntilRest > 0 {
		normalized.DifficultyPenaltyUntilRest = 0
	}
	normalized.FocusTargetID = strings.TrimSpace(normalized.FocusTargetID)
	normalized.ActiveBeastform = NormalizedActiveBeastformPtr(normalized.ActiveBeastform)
	if normalized.StrangePatternsNumber < 0 {
		normalized.StrangePatternsNumber = 0
	}
	normalized.RallyDice = NormalizedDiceValues(normalized.RallyDice)
	normalized.PrayerDice = NormalizedDiceValues(normalized.PrayerDice)
	if normalized.Unstoppable.CurrentValue < 0 {
		normalized.Unstoppable.CurrentValue = 0
	}
	if normalized.Unstoppable.DieSides < 0 {
		normalized.Unstoppable.DieSides = 0
	}
	return normalized
}

// IsZero reports whether the class state carries any mutable runtime data.
func (s CharacterClassState) IsZero() bool {
	normalized := s.Normalized()
	return normalized.AttackBonusUntilRest == 0 &&
		normalized.EvasionBonusUntilHitOrRest == 0 &&
		normalized.DifficultyPenaltyUntilRest == 0 &&
		normalized.FocusTargetID == "" &&
		normalized.ActiveBeastform == nil &&
		normalized.StrangePatternsNumber == 0 &&
		len(normalized.RallyDice) == 0 &&
		len(normalized.PrayerDice) == 0 &&
		!normalized.Unstoppable.Active &&
		normalized.Unstoppable.CurrentValue == 0 &&
		normalized.Unstoppable.DieSides == 0 &&
		!normalized.Unstoppable.UsedThisLongRest &&
		!normalized.ChannelRawPowerUsedThisLongRest
}

// NormalizedDiceValues filters out non-positive values from a dice-value slice.
func NormalizedDiceValues(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	items := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		items = append(items, value)
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

// NormalizedDamageDice filters out invalid damage-die specs from a slice.
func NormalizedDamageDice(values []CharacterDamageDie) []CharacterDamageDie {
	if len(values) == 0 {
		return nil
	}
	items := make([]CharacterDamageDie, 0, len(values))
	for _, value := range values {
		if value.Count <= 0 || value.Sides <= 0 {
			continue
		}
		items = append(items, CharacterDamageDie{Count: value.Count, Sides: value.Sides})
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

// NormalizedActiveBeastformPtr normalizes a beastform pointer, returning nil
// when the beastform ID is empty or whitespace-only.
func NormalizedActiveBeastformPtr(value *CharacterActiveBeastformState) *CharacterActiveBeastformState {
	if value == nil {
		return nil
	}
	normalized := *value
	normalized.BeastformID = strings.TrimSpace(normalized.BeastformID)
	normalized.BaseTrait = strings.TrimSpace(normalized.BaseTrait)
	normalized.AttackTrait = strings.TrimSpace(normalized.AttackTrait)
	normalized.AttackRange = strings.TrimSpace(normalized.AttackRange)
	normalized.DamageType = strings.TrimSpace(normalized.DamageType)
	normalized.EvolutionTraitOverride = strings.TrimSpace(normalized.EvolutionTraitOverride)
	if normalized.TraitBonus < 0 {
		normalized.TraitBonus = 0
	}
	if normalized.EvasionBonus < 0 {
		normalized.EvasionBonus = 0
	}
	if normalized.DamageBonus < 0 {
		normalized.DamageBonus = 0
	}
	normalized.DamageDice = NormalizedDamageDice(normalized.DamageDice)
	if normalized.BeastformID == "" {
		return nil
	}
	return &normalized
}

// WithActiveBeastform returns a normalized copy of the class state with the
// active beastform snapshot replaced.
func WithActiveBeastform(state CharacterClassState, active *CharacterActiveBeastformState) CharacterClassState {
	next := state
	next.ActiveBeastform = NormalizedActiveBeastformPtr(active)
	return next.Normalized()
}
