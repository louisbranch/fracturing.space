// Structured Daggerheart conditions are shared across transport and snapshot
// packages.
package rules

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	ConditionHidden     = "hidden"
	ConditionRestrained = "restrained"
	ConditionVulnerable = "vulnerable"
	ConditionCloaked    = "cloaked"
)

type ConditionClass string

const (
	ConditionClassStandard ConditionClass = "standard"
	ConditionClassTag      ConditionClass = "tag"
	ConditionClassSpecial  ConditionClass = "special"
)

type ConditionClearTrigger string

const (
	ConditionClearTriggerShortRest   ConditionClearTrigger = "short_rest"
	ConditionClearTriggerLongRest    ConditionClearTrigger = "long_rest"
	ConditionClearTriggerSessionEnd  ConditionClearTrigger = "session_end"
	ConditionClearTriggerDamageTaken ConditionClearTrigger = "damage_taken"
)

type ConditionState struct {
	ID            string                  `json:"id"`
	Class         ConditionClass          `json:"class,omitempty"`
	Standard      string                  `json:"standard,omitempty"`
	Code          string                  `json:"code,omitempty"`
	Label         string                  `json:"label,omitempty"`
	Source        string                  `json:"source,omitempty"`
	SourceID      string                  `json:"source_id,omitempty"`
	ClearTriggers []ConditionClearTrigger `json:"clear_triggers,omitempty"`
}

var (
	standardConditionOrder = map[string]int{
		ConditionHidden:     1,
		ConditionRestrained: 2,
		ConditionVulnerable: 3,
		ConditionCloaked:    4,
	}
	standardConditionLabel = map[string]string{
		ConditionHidden:     "Hidden",
		ConditionRestrained: "Restrained",
		ConditionVulnerable: "Vulnerable",
		ConditionCloaked:    "Cloaked",
	}
)

func (c *ConditionState) UnmarshalJSON(data []byte) error {
	if c == nil {
		return fmt.Errorf("condition state is required")
	}
	type rawConditionState ConditionState
	var raw rawConditionState
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = ConditionState(raw)
	return nil
}

func StandardConditionState(code string, options ...func(*ConditionState)) (ConditionState, error) {
	normalizedCode, ok := normalizeStandardConditionCode(code)
	if !ok {
		return ConditionState{}, fmt.Errorf("condition %q is not supported", strings.TrimSpace(code))
	}
	state := ConditionState{
		ID:       normalizedCode,
		Class:    ConditionClassStandard,
		Standard: normalizedCode,
		Code:     normalizedCode,
		Label:    standardConditionLabel[normalizedCode],
	}
	for _, option := range options {
		if option != nil {
			option(&state)
		}
	}
	return normalizeConditionState(state)
}

func WithConditionSource(source, sourceID string) func(*ConditionState) {
	return func(state *ConditionState) {
		if state == nil {
			return
		}
		state.Source = source
		state.SourceID = sourceID
	}
}

func WithConditionClearTriggers(triggers ...ConditionClearTrigger) func(*ConditionState) {
	return func(state *ConditionState) {
		if state == nil {
			return
		}
		state.ClearTriggers = append([]ConditionClearTrigger(nil), triggers...)
	}
}

func NormalizeConditionStates(values []ConditionState) ([]ConditionState, error) {
	if len(values) == 0 {
		return []ConditionState{}, nil
	}

	normalized := make([]ConditionState, 0, len(values))
	seenIDs := make(map[string]struct{}, len(values))
	seenStandard := make(map[string]struct{}, len(values))
	for _, value := range values {
		current, err := normalizeConditionState(value)
		if err != nil {
			return nil, err
		}
		if _, ok := seenIDs[current.ID]; ok {
			continue
		}
		if current.Class == ConditionClassStandard {
			if _, ok := seenStandard[current.Standard]; ok {
				continue
			}
			seenStandard[current.Standard] = struct{}{}
		}
		seenIDs[current.ID] = struct{}{}
		normalized = append(normalized, current)
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		left := normalized[i]
		right := normalized[j]
		if left.Class != right.Class {
			return left.Class < right.Class
		}
		if left.Class == ConditionClassStandard {
			return standardConditionOrder[left.Standard] < standardConditionOrder[right.Standard]
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		return left.ID < right.ID
	})

	return normalized, nil
}

func ConditionStatesEqual(left, right []ConditionState) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !conditionStatesMatch(left[i], right[i]) {
			return false
		}
	}
	return true
}

func NormalizeConditions(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{}, nil
	}
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		current, ok := normalizeStandardConditionCode(value)
		if !ok {
			return nil, fmt.Errorf("condition %q is not supported", strings.TrimSpace(value))
		}
		if _, exists := seen[current]; exists {
			continue
		}
		seen[current] = struct{}{}
		normalized = append(normalized, current)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return standardConditionOrder[normalized[i]] < standardConditionOrder[normalized[j]]
	})
	return normalized, nil
}

func ConditionsEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func DiffConditions(before, after []string) (added []string, removed []string) {
	beforeSet := make(map[string]struct{}, len(before))
	for _, value := range before {
		beforeSet[value] = struct{}{}
	}
	afterSet := make(map[string]struct{}, len(after))
	for _, value := range after {
		afterSet[value] = struct{}{}
		if _, ok := beforeSet[value]; !ok {
			added = append(added, value)
		}
	}
	for _, value := range before {
		if _, ok := afterSet[value]; !ok {
			removed = append(removed, value)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

func DiffConditionStates(before, after []ConditionState) (added []ConditionState, removed []ConditionState) {
	beforeByID := make(map[string]ConditionState, len(before))
	for _, value := range before {
		beforeByID[value.ID] = value
	}
	afterByID := make(map[string]ConditionState, len(after))
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

func ClearConditionStatesByTrigger(values []ConditionState, trigger ConditionClearTrigger) (remaining []ConditionState, removed []ConditionState) {
	if len(values) == 0 || trigger == "" {
		return append([]ConditionState(nil), values...), nil
	}
	for _, value := range values {
		if conditionHasTrigger(value, trigger) {
			removed = append(removed, value)
			continue
		}
		remaining = append(remaining, value)
	}
	return remaining, removed
}

func HasConditionCode(values []ConditionState, code string) bool {
	normalizedCode, ok := normalizeStandardConditionCode(code)
	if ok {
		code = normalizedCode
	}
	code = strings.TrimSpace(strings.ToLower(code))
	if code == "" {
		return false
	}
	for _, value := range values {
		if value.Code == code || value.Standard == code || value.ID == code {
			return true
		}
	}
	return false
}

func RemoveConditionCode(values []ConditionState, code string) []ConditionState {
	normalizedCode, ok := normalizeStandardConditionCode(code)
	if ok {
		code = normalizedCode
	}
	code = strings.TrimSpace(strings.ToLower(code))
	if code == "" {
		return append([]ConditionState(nil), values...)
	}
	result := make([]ConditionState, 0, len(values))
	for _, value := range values {
		if value.Code == code || value.Standard == code || value.ID == code {
			continue
		}
		result = append(result, value)
	}
	return result
}

func ConditionCodes(values []ConditionState) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value.Code != "" {
			result = append(result, value.Code)
			continue
		}
		if value.Standard != "" {
			result = append(result, value.Standard)
			continue
		}
		result = append(result, value.ID)
	}
	return result
}

func normalizeConditionState(value ConditionState) (ConditionState, error) {
	normalized := value
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.Class = ConditionClass(strings.ToLower(strings.TrimSpace(string(normalized.Class))))
	normalized.Standard = strings.ToLower(strings.TrimSpace(normalized.Standard))
	normalized.Code = strings.ToLower(strings.TrimSpace(normalized.Code))
	normalized.Label = strings.TrimSpace(normalized.Label)
	normalized.Source = strings.TrimSpace(normalized.Source)
	normalized.SourceID = strings.TrimSpace(normalized.SourceID)
	normalized.ClearTriggers = normalizeConditionClearTriggers(normalized.ClearTriggers)

	if normalized.Class == "" {
		if normalized.Standard != "" {
			normalized.Class = ConditionClassStandard
		} else if normalized.Code != "" {
			normalized.Class = ConditionClassSpecial
		}
	}

	switch normalized.Class {
	case ConditionClassStandard:
		code, ok := normalizeStandardConditionCode(firstNonEmpty(normalized.Standard, normalized.Code, normalized.ID))
		if !ok {
			return ConditionState{}, fmt.Errorf("condition %q is not supported", firstNonEmpty(normalized.Standard, normalized.Code, normalized.ID))
		}
		normalized.Standard = code
		normalized.Code = code
		if normalized.ID == "" {
			normalized.ID = code
		}
		if normalized.Label == "" {
			normalized.Label = standardConditionLabel[code]
		}
	case ConditionClassTag, ConditionClassSpecial:
		if normalized.Code == "" {
			normalized.Code = strings.ToLower(strings.TrimSpace(firstNonEmpty(normalized.Label, normalized.ID)))
		}
		if normalized.Code == "" {
			return ConditionState{}, fmt.Errorf("condition code is required")
		}
		if normalized.ID == "" {
			normalized.ID = normalized.Code
		}
		if normalized.Label == "" {
			normalized.Label = normalized.Code
		}
		normalized.Standard = ""
	default:
		return ConditionState{}, fmt.Errorf("condition class %q is invalid", normalized.Class)
	}

	if normalized.ID == "" {
		return ConditionState{}, fmt.Errorf("condition id is required")
	}
	return normalized, nil
}

func normalizeConditionClearTriggers(values []ConditionClearTrigger) []ConditionClearTrigger {
	if len(values) == 0 {
		return nil
	}
	result := make([]ConditionClearTrigger, 0, len(values))
	seen := make(map[ConditionClearTrigger]struct{}, len(values))
	for _, value := range values {
		current := ConditionClearTrigger(strings.ToLower(strings.TrimSpace(string(value))))
		switch current {
		case ConditionClearTriggerShortRest, ConditionClearTriggerLongRest, ConditionClearTriggerSessionEnd, ConditionClearTriggerDamageTaken:
		default:
			continue
		}
		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}
		result = append(result, current)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func conditionHasTrigger(value ConditionState, trigger ConditionClearTrigger) bool {
	for _, current := range value.ClearTriggers {
		if current == trigger {
			return true
		}
	}
	return false
}

func conditionStatesMatch(left, right ConditionState) bool {
	if left.ID != right.ID ||
		left.Class != right.Class ||
		left.Standard != right.Standard ||
		left.Code != right.Code ||
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

func normalizeStandardConditionCode(code string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(code))
	if _, ok := standardConditionOrder[normalized]; !ok {
		return "", false
	}
	return normalized, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
