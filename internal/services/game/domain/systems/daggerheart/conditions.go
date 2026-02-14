package daggerheart

import (
	"fmt"
	"sort"
	"strings"
)

// Condition identifiers for Daggerheart character conditions.
const (
	ConditionHidden     = "hidden"
	ConditionRestrained = "restrained"
	ConditionVulnerable = "vulnerable"
)

var conditionOrder = map[string]int{
	ConditionHidden:     1,
	ConditionRestrained: 2,
	ConditionVulnerable: 3,
}

// NormalizeConditions validates and returns a canonical condition list.
func NormalizeConditions(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{}, nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, fmt.Errorf("condition must not be empty")
		}
		lowered := strings.ToLower(trimmed)
		if _, ok := conditionOrder[lowered]; !ok {
			return nil, fmt.Errorf("condition %q is not supported", trimmed)
		}
		if _, ok := seen[lowered]; ok {
			continue
		}
		seen[lowered] = struct{}{}
		normalized = append(normalized, lowered)
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		return conditionOrder[normalized[i]] < conditionOrder[normalized[j]]
	})

	return normalized, nil
}

// DiffConditions returns added and removed condition lists between two sets.
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
	return added, removed
}

// ConditionsEqual compares two condition lists for equality.
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
