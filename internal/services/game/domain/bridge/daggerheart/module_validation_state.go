package daggerheart

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func validateGMFearSetPayload(raw json.RawMessage) error {
	var payload GMFearSetPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if payload.After == nil {
		return errors.New("after is required")
	}
	if err := requireRange(*payload.After, GMFearMin, GMFearMax, "after"); err != nil {
		return err
	}
	return nil
}

func validateGMFearChangedPayload(raw json.RawMessage) error {
	var payload GMFearChangedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if payload.After < GMFearMin || payload.After > GMFearMax {
		return fmt.Errorf("after must be in range %d..%d", GMFearMin, GMFearMax)
	}
	if payload.Before == payload.After {
		return errors.New("before and after must differ")
	}
	return nil
}

func validateCharacterStatePatchPayload(raw json.RawMessage) error {
	var payload CharacterStatePatchPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if !hasCharacterStateChange(payload) {
		return errors.New("character_state patch must change at least one field")
	}
	return nil
}

func validateCharacterStatePatchedPayload(raw json.RawMessage) error {
	return validateCharacterStatePatchPayload(raw)
}

func validateHopeSpendPayload(raw json.RawMessage) error {
	var payload HopeSpendPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if payload.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if payload.Before == payload.After {
		return errors.New("before and after must differ")
	}
	if abs(payload.Before-payload.After) != payload.Amount {
		return errors.New("amount must match before and after delta")
	}
	return nil
}

func validateStressSpendPayload(raw json.RawMessage) error {
	var payload StressSpendPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	if payload.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if payload.Before == payload.After {
		return errors.New("before and after must differ")
	}
	if abs(payload.Before-payload.After) != payload.Amount {
		return errors.New("amount must match before and after delta")
	}
	return nil
}

func validateConditionChangePayload(raw json.RawMessage) error {
	var payload ConditionChangePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID) == "" {
		return errors.New("character_id is required")
	}
	return validateConditionSetPayload(
		payload.ConditionsBefore,
		payload.ConditionsAfter,
		payload.Added,
		payload.Removed,
	)
}

func validateConditionChangedPayload(raw json.RawMessage) error {
	return validateConditionChangePayload(raw)
}

func validateLoadoutSwapPayload(raw json.RawMessage) error {
	var payload LoadoutSwapPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.CharacterID, "character_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.CardID, "card_id"); err != nil {
		return err
	}
	return nil
}

func validateLoadoutSwappedPayload(raw json.RawMessage) error {
	return validateLoadoutSwapPayload(raw)
}

func validateRestTakePayload(raw json.RawMessage) error {
	var payload RestTakePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.RestType, "rest_type"); err != nil {
		return err
	}
	if payload.LongTermCountdown != nil {
		if err := validateRestLongTermCountdownPayload(*payload.LongTermCountdown); err != nil {
			return err
		}
	}
	if !hasRestTakeMutation(payload) {
		return errors.New("rest.take must change at least one field")
	}
	return nil
}

func validateRestTakenPayload(raw json.RawMessage) error {
	return validateRestTakePayload(raw)
}

func validateCountdownCreatePayload(raw json.RawMessage) error {
	var payload CountdownCreatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.CountdownID, "countdown_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.Name, "name"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.Kind, "kind"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.Direction, "direction"); err != nil {
		return err
	}
	if err := requirePositive(payload.Max, "max"); err != nil {
		return err
	}
	if payload.Current < 0 || payload.Current > payload.Max {
		return fmt.Errorf("current must be in range 0..%d", payload.Max)
	}
	variant := strings.TrimSpace(payload.Variant)
	if variant == "" {
		variant = "standard"
	}
	switch variant {
	case "standard", "dynamic", "linked":
		// valid
	default:
		return fmt.Errorf("unknown countdown variant %q; must be standard, dynamic, or linked", variant)
	}
	if variant == "dynamic" && strings.TrimSpace(payload.TriggerEventType) == "" {
		return errors.New("trigger_event_type is required for dynamic countdowns")
	}
	if variant == "linked" && strings.TrimSpace(payload.LinkedCountdownID) == "" {
		return errors.New("linked_countdown_id is required for linked countdowns")
	}
	return nil
}

func validateCountdownCreatedPayload(raw json.RawMessage) error {
	return validateCountdownCreatePayload(raw)
}

func validateCountdownUpdatePayload(raw json.RawMessage) error {
	var payload CountdownUpdatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CountdownID) == "" {
		return errors.New("countdown_id is required")
	}
	if payload.Before == payload.After && payload.Delta == 0 {
		return errors.New("countdown update must change value")
	}
	return nil
}

func validateCountdownUpdatedPayload(raw json.RawMessage) error {
	return validateCountdownUpdatePayload(raw)
}

func validateCountdownDeletePayload(raw json.RawMessage) error {
	var payload CountdownDeletePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CountdownID) == "" {
		return errors.New("countdown_id is required")
	}
	return nil
}

func validateCountdownDeletedPayload(raw json.RawMessage) error {
	return validateCountdownDeletePayload(raw)
}

func validateAdversaryConditionChangePayload(raw json.RawMessage) error {
	var payload AdversaryConditionChangePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID) == "" {
		return errors.New("adversary_id is required")
	}
	return validateConditionSetPayload(
		payload.ConditionsBefore,
		payload.ConditionsAfter,
		payload.Added,
		payload.Removed,
	)
}

func validateAdversaryConditionChangedPayload(raw json.RawMessage) error {
	return validateAdversaryConditionChangePayload(raw)
}

func validateConditionSetPayload(before, after, added, removed []string) error {
	normalizedAfter, _, err := normalizeConditionListField(after, "conditions_after", true)
	if err != nil {
		return err
	}

	normalizedBefore, hasBefore, err := normalizeConditionListField(before, "conditions_before", false)
	if err != nil {
		return err
	}
	normalizedAdded, hasAdded, err := normalizeConditionListField(added, "added", false)
	if err != nil {
		return err
	}
	normalizedRemoved, hasRemoved, err := normalizeConditionListField(removed, "removed", false)
	if err != nil {
		return err
	}

	expectedAdded := normalizedAfter
	expectedRemoved := []string{}
	if hasBefore {
		expectedAdded, expectedRemoved = DiffConditions(normalizedBefore, normalizedAfter)
	}

	if !hasBefore && hasRemoved && len(normalizedRemoved) > 0 {
		return errors.New("conditions_before is required when removed are provided")
	}

	if hasAdded {
		if !ConditionsEqual(normalizedAdded, expectedAdded) {
			if hasBefore {
				return errors.New("added must match conditions_before and conditions_after diff")
			}
			return errors.New("added must match conditions_after when conditions_before is omitted")
		}
	}

	if hasRemoved && !ConditionsEqual(normalizedRemoved, expectedRemoved) {
		if hasBefore {
			return errors.New("removed must match conditions_before and conditions_after diff")
		}
		return errors.New("removed must be empty when conditions_before is omitted")
	}

	if hasBefore {
		if ConditionsEqual(normalizedBefore, normalizedAfter) &&
			len(normalizedAdded) == 0 && len(normalizedRemoved) == 0 {
			return errors.New("conditions must change")
		}
	} else if len(normalizedAfter) == 0 && len(normalizedAdded) == 0 && len(normalizedRemoved) == 0 {
		return errors.New("conditions must change")
	}

	return nil
}

func normalizeConditionListField(values []string, field string, required bool) ([]string, bool, error) {
	if values == nil {
		if required {
			return nil, false, fmt.Errorf("%s is required", field)
		}
		return nil, false, nil
	}

	normalized, err := NormalizeConditions(values)
	if err != nil {
		return nil, true, fmt.Errorf("%s: %w", field, err)
	}
	return normalized, true, nil
}

func hasCharacterStateChange(payload CharacterStatePatchPayload) bool {
	return hasIntFieldChange(payload.HPBefore, payload.HPAfter) ||
		hasIntFieldChange(payload.HopeBefore, payload.HopeAfter) ||
		hasIntFieldChange(payload.HopeMaxBefore, payload.HopeMaxAfter) ||
		hasIntFieldChange(payload.StressBefore, payload.StressAfter) ||
		hasIntFieldChange(payload.ArmorBefore, payload.ArmorAfter) ||
		hasStringFieldChange(payload.LifeStateBefore, payload.LifeStateAfter)
}

func hasConditionListMutation(before, after []string) bool {
	beforeNormalized, err := NormalizeConditions(before)
	if err != nil {
		return true
	}
	afterNormalized, err := NormalizeConditions(after)
	if err != nil {
		return true
	}
	return !ConditionsEqual(beforeNormalized, afterNormalized)
}

func hasRestCharacterStateMutation(payload RestCharacterStatePatch) bool {
	return hasIntFieldChange(payload.HopeBefore, payload.HopeAfter) ||
		hasIntFieldChange(payload.StressBefore, payload.StressAfter) ||
		hasIntFieldChange(payload.ArmorBefore, payload.ArmorAfter)
}

func hasRestTakeMutation(payload RestTakePayload) bool {
	if payload.GMFearBefore != payload.GMFearAfter ||
		payload.ShortRestsBefore != payload.ShortRestsAfter ||
		payload.RefreshRest ||
		payload.RefreshLongRest ||
		payload.LongTermCountdown != nil {
		return true
	}
	for _, patch := range payload.CharacterStates {
		if hasRestCharacterStateMutation(patch) {
			return true
		}
	}
	return false
}

func validateRestLongTermCountdownPayload(payload CountdownUpdatePayload) error {
	if strings.TrimSpace(payload.CountdownID) == "" {
		return errors.New("long_term_countdown.countdown_id is required")
	}
	if payload.Before == payload.After && payload.Delta == 0 {
		return errors.New("long_term_countdown must change value")
	}
	return nil
}

func hasIntFieldChange(before, after *int) bool {
	if after == nil {
		return false
	}
	if before == nil {
		return true
	}
	return *before != *after
}

func hasStringFieldChange(before, after *string) bool {
	if after == nil {
		return false
	}
	if before == nil {
		return true
	}
	return *before != *after
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
