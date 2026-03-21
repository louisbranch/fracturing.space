// Stat modifier domain types for runtime character stat adjustments
// (evasion, thresholds, proficiency, armor score).
//
// Mirrors ConditionState lifecycle: GM adds/removes modifiers; ClearTriggers
// control automatic removal on rest/damage/session boundaries.
package rules

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// StatModifierTarget identifies which stat a modifier adjusts. This covers
// both derived combat stats (evasion, thresholds) and base character traits
// (strength, finesse, etc.) used in action/reaction rolls.
type StatModifierTarget string

const (
	// Derived combat stats.
	StatModifierTargetEvasion         StatModifierTarget = "evasion"
	StatModifierTargetMajorThreshold  StatModifierTarget = "major_threshold"
	StatModifierTargetSevereThreshold StatModifierTarget = "severe_threshold"
	StatModifierTargetProficiency     StatModifierTarget = "proficiency"
	StatModifierTargetArmorScore      StatModifierTarget = "armor_score"

	// Base character traits.
	StatModifierTargetStrength  StatModifierTarget = "strength"
	StatModifierTargetFinesse   StatModifierTarget = "finesse"
	StatModifierTargetAgility   StatModifierTarget = "agility"
	StatModifierTargetInstinct  StatModifierTarget = "instinct"
	StatModifierTargetPresence  StatModifierTarget = "presence"
	StatModifierTargetKnowledge StatModifierTarget = "knowledge"
)

// ValidStatModifierTarget reports whether target is a recognized stat modifier
// target string.
func ValidStatModifierTarget(target string) bool {
	switch StatModifierTarget(strings.ToLower(strings.TrimSpace(target))) {
	case StatModifierTargetEvasion,
		StatModifierTargetMajorThreshold,
		StatModifierTargetSevereThreshold,
		StatModifierTargetProficiency,
		StatModifierTargetArmorScore,
		StatModifierTargetStrength,
		StatModifierTargetFinesse,
		StatModifierTargetAgility,
		StatModifierTargetInstinct,
		StatModifierTargetPresence,
		StatModifierTargetKnowledge:
		return true
	default:
		return false
	}
}

// StatModifierState captures a single stat modifier applied to a character.
type StatModifierState struct {
	ID            string                  `json:"id"`
	Target        StatModifierTarget      `json:"target"`
	Delta         int                     `json:"delta"`
	Label         string                  `json:"label,omitempty"`
	Source        string                  `json:"source,omitempty"`
	SourceID      string                  `json:"source_id,omitempty"`
	ClearTriggers []ConditionClearTrigger `json:"clear_triggers,omitempty"`
}

// NormalizeStatModifiers validates and deduplicates a slice of stat modifiers,
// returning them in stable order (by target then ID).
func NormalizeStatModifiers(values []StatModifierState) ([]StatModifierState, error) {
	if len(values) == 0 {
		return []StatModifierState{}, nil
	}
	normalized := make([]StatModifierState, 0, len(values))
	seenIDs := make(map[string]struct{}, len(values))
	for _, value := range values {
		current, err := normalizeStatModifier(value)
		if err != nil {
			return nil, err
		}
		if _, ok := seenIDs[current.ID]; ok {
			continue
		}
		seenIDs[current.ID] = struct{}{}
		normalized = append(normalized, current)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].Target != normalized[j].Target {
			return normalized[i].Target < normalized[j].Target
		}
		return normalized[i].ID < normalized[j].ID
	})
	return normalized, nil
}

// StatModifiersEqual reports whether two stat modifier slices are identical.
func StatModifiersEqual(left, right []StatModifierState) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !statModifiersMatch(left[i], right[i]) {
			return false
		}
	}
	return true
}

// DiffStatModifiers returns the added and removed modifiers between before and
// after slices (matched by ID).
func DiffStatModifiers(before, after []StatModifierState) (added []StatModifierState, removed []StatModifierState) {
	beforeByID := make(map[string]StatModifierState, len(before))
	for _, value := range before {
		beforeByID[value.ID] = value
	}
	afterByID := make(map[string]StatModifierState, len(after))
	for _, value := range after {
		afterByID[value.ID] = value
		if _, ok := beforeByID[value.ID]; !ok {
			added = append(added, value)
		}
	}
	for _, value := range before {
		if _, ok := afterByID[value.ID]; !ok {
			removed = append(removed, value)
		}
	}
	return added, removed
}

// ClearStatModifiersByTrigger removes modifiers whose ClearTriggers include the
// given trigger, returning remaining and removed slices.
func ClearStatModifiersByTrigger(values []StatModifierState, trigger ConditionClearTrigger) (remaining []StatModifierState, removed []StatModifierState) {
	if len(values) == 0 || trigger == "" {
		return append([]StatModifierState(nil), values...), nil
	}
	for _, value := range values {
		if statModifierHasTrigger(value, trigger) {
			removed = append(removed, value)
			continue
		}
		remaining = append(remaining, value)
	}
	return remaining, removed
}

// --- internal helpers ---

func normalizeStatModifier(value StatModifierState) (StatModifierState, error) {
	normalized := value
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.Target = StatModifierTarget(strings.ToLower(strings.TrimSpace(string(normalized.Target))))
	normalized.Label = strings.TrimSpace(normalized.Label)
	normalized.Source = strings.TrimSpace(normalized.Source)
	normalized.SourceID = strings.TrimSpace(normalized.SourceID)
	normalized.ClearTriggers = normalizeConditionClearTriggers(normalized.ClearTriggers)

	if normalized.ID == "" {
		return StatModifierState{}, fmt.Errorf("stat modifier id is required")
	}
	if !ValidStatModifierTarget(string(normalized.Target)) {
		return StatModifierState{}, fmt.Errorf("stat modifier target %q is not supported", normalized.Target)
	}
	return normalized, nil
}

func statModifiersMatch(left, right StatModifierState) bool {
	if left.ID != right.ID ||
		left.Target != right.Target ||
		left.Delta != right.Delta ||
		left.Label != right.Label ||
		left.Source != right.Source ||
		left.SourceID != right.SourceID ||
		len(left.ClearTriggers) != len(right.ClearTriggers) {
		return false
	}
	for i := range left.ClearTriggers {
		if left.ClearTriggers[i] != right.ClearTriggers[i] {
			return false
		}
	}
	return true
}

func statModifierHasTrigger(value StatModifierState, trigger ConditionClearTrigger) bool {
	return slices.Contains(value.ClearTriggers, trigger)
}
