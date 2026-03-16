package daggerheart

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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
	if payload.Value < GMFearMin || payload.Value > GMFearMax {
		return fmt.Errorf("value must be in range %d..%d", GMFearMin, GMFearMax)
	}
	return nil
}

func validateGMMoveApplyPayload(raw json.RawMessage) error {
	var payload GMMoveApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if payload.FearSpent <= 0 {
		return errors.New("fear_spent must be greater than zero")
	}
	return validateGMMoveTarget(payload.Target)
}

func validateGMMoveAppliedPayload(raw json.RawMessage) error {
	var payload GMMoveAppliedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if payload.FearSpent <= 0 {
		return errors.New("fear_spent must be greater than zero")
	}
	return validateGMMoveTarget(payload.Target)
}

func validateGMMoveTarget(target GMMoveTarget) error {
	targetType, ok := NormalizeGMMoveTargetType(string(target.Type))
	if !ok {
		return errors.New("target type is unsupported")
	}
	switch targetType {
	case GMMoveTargetTypeDirectMove:
		if _, ok := NormalizeGMMoveKind(string(target.Kind)); !ok {
			return errors.New("kind is unsupported")
		}
		if _, ok := NormalizeGMMoveShape(string(target.Shape)); !ok {
			return errors.New("shape is unsupported")
		}
		if target.Shape == GMMoveShapeCustom && strings.TrimSpace(target.Description) == "" {
			return errors.New("description is required for custom shape")
		}
		if target.Shape == GMMoveShapeSpotlightAdversary && strings.TrimSpace(target.AdversaryID.String()) == "" {
			return errors.New("adversary_id is required for spotlight_adversary")
		}
	case GMMoveTargetTypeAdversaryFeature:
		if strings.TrimSpace(target.AdversaryID.String()) == "" {
			return errors.New("adversary_id is required")
		}
		if strings.TrimSpace(target.FeatureID) == "" {
			return errors.New("feature_id is required")
		}
	case GMMoveTargetTypeEnvironmentFeature:
		if strings.TrimSpace(target.EnvironmentEntityID.String()) == "" && strings.TrimSpace(target.EnvironmentID) == "" {
			return errors.New("environment_entity_id is required")
		}
		if strings.TrimSpace(target.FeatureID) == "" {
			return errors.New("feature_id is required")
		}
	case GMMoveTargetTypeAdversaryExperience:
		if strings.TrimSpace(target.AdversaryID.String()) == "" {
			return errors.New("adversary_id is required")
		}
		if strings.TrimSpace(target.ExperienceName) == "" {
			return errors.New("experience_name is required")
		}
	default:
		return errors.New("target type is unsupported")
	}
	return nil
}

func validateCharacterProfileReplacePayload(raw json.RawMessage) error {
	var payload CharacterProfileReplacePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.CharacterID.String(), "character_id"); err != nil {
		return err
	}
	return payload.Profile.Validate()
}

func validateCharacterProfileReplacedPayload(raw json.RawMessage) error {
	var payload CharacterProfileReplacedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.CharacterID.String(), "character_id"); err != nil {
		return err
	}
	return payload.Profile.Validate()
}

func validateCharacterProfileDeletePayload(raw json.RawMessage) error {
	var payload CharacterProfileDeletePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return requireTrimmedValue(payload.CharacterID.String(), "character_id")
}

func validateCharacterProfileDeletedPayload(raw json.RawMessage) error {
	var payload CharacterProfileDeletedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	return requireTrimmedValue(payload.CharacterID.String(), "character_id")
}

func validateCharacterStatePatchPayload(raw json.RawMessage) error {
	var payload CharacterStatePatchPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if !hasCharacterStateChange(payload) {
		return errors.New("character_state patch must change at least one field")
	}
	return nil
}

func validateCharacterStatePatchedPayload(raw json.RawMessage) error {
	var payload CharacterStatePatchedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if payload.HP == nil && payload.Hope == nil && payload.HopeMax == nil &&
		payload.Stress == nil && payload.Armor == nil && payload.LifeState == nil &&
		payload.ClassState == nil && payload.SubclassState == nil && payload.ImpenetrableUsedThisShortRest == nil {
		return errors.New("character_state_patched must include at least one after field")
	}
	return nil
}

func validateClassFeatureApplyPayload(raw json.RawMessage) error {
	var payload ClassFeatureApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(payload.Feature) == "" {
		return errors.New("feature is required")
	}
	if len(payload.Targets) == 0 {
		return errors.New("class_feature apply requires at least one target")
	}
	for _, target := range payload.Targets {
		if strings.TrimSpace(target.CharacterID.String()) == "" {
			return errors.New("class_feature apply target character_id is required")
		}
		if !hasIntFieldChange(target.HPBefore, target.HPAfter) &&
			!hasIntFieldChange(target.HopeBefore, target.HopeAfter) &&
			!hasIntFieldChange(target.ArmorBefore, target.ArmorAfter) &&
			!hasClassStateFieldChange(target.ClassStateBefore, target.ClassStateAfter) {
			return errors.New("class_feature apply must change at least one field per target")
		}
	}
	return nil
}

func validateSubclassFeatureApplyPayload(raw json.RawMessage) error {
	var payload SubclassFeatureApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(payload.Feature) == "" {
		return errors.New("feature is required")
	}
	if len(payload.Targets) == 0 && len(payload.CharacterConditionTargets) == 0 && len(payload.AdversaryConditionTargets) == 0 {
		return errors.New("subclass_feature apply requires at least one consequence")
	}
	hasMutation := len(payload.CharacterConditionTargets) > 0 || len(payload.AdversaryConditionTargets) > 0
	for _, target := range payload.Targets {
		if strings.TrimSpace(target.CharacterID.String()) == "" {
			return errors.New("subclass_feature apply target character_id is required")
		}
		if !hasIntFieldChange(target.HPBefore, target.HPAfter) &&
			!hasIntFieldChange(target.HopeBefore, target.HopeAfter) &&
			!hasIntFieldChange(target.StressBefore, target.StressAfter) &&
			!hasIntFieldChange(target.ArmorBefore, target.ArmorAfter) &&
			!hasClassStateFieldChange(target.ClassStateBefore, target.ClassStateAfter) &&
			!hasSubclassStateFieldChange(target.SubclassStateBefore, target.SubclassStateAfter) {
			continue
		}
		hasMutation = true
	}
	if !hasMutation {
		return errors.New("subclass_feature apply must change at least one field")
	}
	return nil
}

func validateBeastformTransformPayload(raw json.RawMessage) error {
	var payload BeastformTransformPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.BeastformID) == "" {
		return errors.New("beastform_id is required")
	}
	if !hasIntFieldChange(payload.HopeBefore, payload.HopeAfter) &&
		!hasIntFieldChange(payload.StressBefore, payload.StressAfter) &&
		!hasClassStateFieldChange(payload.ClassStateBefore, payload.ClassStateAfter) {
		return errors.New("beastform transform must change at least one field")
	}
	return nil
}

func validateBeastformDropPayload(raw json.RawMessage) error {
	var payload BeastformDropPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.BeastformID) == "" {
		return errors.New("beastform_id is required")
	}
	if !hasClassStateFieldChange(payload.ClassStateBefore, payload.ClassStateAfter) {
		return errors.New("beastform drop must change class state")
	}
	return nil
}

func validateBeastformTransformedPayload(raw json.RawMessage) error {
	var payload BeastformTransformedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.BeastformID) == "" {
		return errors.New("beastform_id is required")
	}
	if normalizedActiveBeastformPtr(payload.ActiveBeastform) == nil {
		return errors.New("active_beastform is required")
	}
	return nil
}

func validateBeastformDroppedPayload(raw json.RawMessage) error {
	var payload BeastformDroppedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.BeastformID) == "" {
		return errors.New("beastform_id is required")
	}
	return nil
}

func validateCompanionExperienceBeginPayload(raw json.RawMessage) error {
	var payload CompanionExperienceBeginPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.ExperienceID) == "" {
		return errors.New("experience_id is required")
	}
	if !hasCompanionStateFieldChange(payload.CompanionStateBefore, payload.CompanionStateAfter) {
		return errors.New("companion begin must change companion state")
	}
	return nil
}

func validateCompanionReturnPayload(raw json.RawMessage) error {
	var payload CompanionReturnPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.Resolution) == "" {
		return errors.New("resolution is required")
	}
	if !hasIntFieldChange(payload.StressBefore, payload.StressAfter) &&
		!hasCompanionStateFieldChange(payload.CompanionStateBefore, payload.CompanionStateAfter) {
		return errors.New("companion return must change at least one field")
	}
	return nil
}

func validateCompanionExperienceBegunPayload(raw json.RawMessage) error {
	var payload CompanionExperienceBegunPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.ExperienceID) == "" {
		return errors.New("experience_id is required")
	}
	if normalizedCompanionStatePtr(payload.CompanionState) == nil {
		return errors.New("companion_state is required")
	}
	return nil
}

func validateCompanionReturnedPayload(raw json.RawMessage) error {
	var payload CompanionReturnedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.Resolution) == "" {
		return errors.New("resolution is required")
	}
	if normalizedCompanionStatePtr(payload.CompanionState) == nil {
		return errors.New("companion_state is required")
	}
	return nil
}

func validateHopeSpendPayload(raw json.RawMessage) error {
	var payload HopeSpendPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
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
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
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
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
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
	var payload ConditionChangedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if payload.Conditions == nil {
		return errors.New("conditions_after is required")
	}
	if _, err := NormalizeConditionStates(payload.Conditions); err != nil {
		return fmt.Errorf("conditions_after: %w", err)
	}
	return nil
}

func validateLoadoutSwapPayload(raw json.RawMessage) error {
	var payload LoadoutSwapPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.CharacterID.String(), "character_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.CardID, "card_id"); err != nil {
		return err
	}
	return nil
}

func validateLoadoutSwappedPayload(raw json.RawMessage) error {
	var payload LoadoutSwappedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.CharacterID.String(), "character_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.CardID, "card_id"); err != nil {
		return err
	}
	return nil
}

func validateRestTakePayload(raw json.RawMessage) error {
	var payload RestTakePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.RestType, "rest_type"); err != nil {
		return err
	}
	if len(payload.Participants) == 0 {
		return errors.New("participants are required")
	}
	for _, participantID := range payload.Participants {
		if err := requireTrimmedValue(participantID.String(), "participants.character_id"); err != nil {
			return err
		}
	}
	for _, update := range payload.CountdownUpdates {
		if err := validateRestLongTermCountdownPayload(update); err != nil {
			return err
		}
	}
	for _, move := range payload.DowntimeMoves {
		if err := validateDowntimeMoveAppliedPayloadFields(move); err != nil {
			return err
		}
	}
	if !hasRestTakeMutation(payload) {
		return errors.New("rest.take must record at least one durable outcome")
	}
	return nil
}

func validateRestTakenPayload(raw json.RawMessage) error {
	var payload RestTakenPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.RestType, "rest_type"); err != nil {
		return err
	}
	if payload.GMFear < GMFearMin || payload.GMFear > GMFearMax {
		return fmt.Errorf("gm_fear_after must be in range %d..%d", GMFearMin, GMFearMax)
	}
	if len(payload.Participants) == 0 {
		return errors.New("participants are required")
	}
	return nil
}

func validateCountdownCreatePayload(raw json.RawMessage) error {
	var payload CountdownCreatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.CountdownID.String(), "countdown_id"); err != nil {
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
	if variant == "linked" && strings.TrimSpace(payload.LinkedCountdownID.String()) == "" {
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
	if strings.TrimSpace(payload.CountdownID.String()) == "" {
		return errors.New("countdown_id is required")
	}
	if payload.Before == payload.After && payload.Delta == 0 {
		return errors.New("countdown update must change value")
	}
	return nil
}

func validateCountdownUpdatedPayload(raw json.RawMessage) error {
	var payload CountdownUpdatedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CountdownID.String()) == "" {
		return errors.New("countdown_id is required")
	}
	return nil
}

func validateCountdownDeletePayload(raw json.RawMessage) error {
	var payload CountdownDeletePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CountdownID.String()) == "" {
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
	if strings.TrimSpace(payload.AdversaryID.String()) == "" {
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
	var payload AdversaryConditionChangedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID.String()) == "" {
		return errors.New("adversary_id is required")
	}
	if payload.Conditions == nil {
		return errors.New("conditions_after is required")
	}
	if _, err := NormalizeConditionStates(payload.Conditions); err != nil {
		return fmt.Errorf("conditions_after: %w", err)
	}
	return nil
}

func validateConditionSetPayload(before, after, added, removed []ConditionState) error {
	normalizedAfter, _, err := normalizeConditionStateListField(after, "conditions_after", true)
	if err != nil {
		return err
	}

	normalizedBefore, hasBefore, err := normalizeConditionStateListField(before, "conditions_before", false)
	if err != nil {
		return err
	}
	normalizedAdded, hasAdded, err := normalizeConditionStateListField(added, "added", false)
	if err != nil {
		return err
	}
	normalizedRemoved, hasRemoved, err := normalizeConditionStateListField(removed, "removed", false)
	if err != nil {
		return err
	}

	expectedAdded := normalizedAfter
	expectedRemoved := []ConditionState{}
	if hasBefore {
		expectedAdded, expectedRemoved = DiffConditionStates(normalizedBefore, normalizedAfter)
	}

	if !hasBefore && hasRemoved && len(normalizedRemoved) > 0 {
		return errors.New("conditions_before is required when removed are provided")
	}

	if hasAdded {
		if !ConditionStatesEqual(normalizedAdded, expectedAdded) {
			if hasBefore {
				return errors.New("added must match conditions_before and conditions_after diff")
			}
			return errors.New("added must match conditions_after when conditions_before is omitted")
		}
	}

	if hasRemoved && !ConditionStatesEqual(normalizedRemoved, expectedRemoved) {
		if hasBefore {
			return errors.New("removed must match conditions_before and conditions_after diff")
		}
		return errors.New("removed must be empty when conditions_before is omitted")
	}

	if hasBefore {
		if ConditionStatesEqual(normalizedBefore, normalizedAfter) &&
			len(normalizedAdded) == 0 && len(normalizedRemoved) == 0 {
			return errors.New("conditions must change")
		}
	} else if len(normalizedAfter) == 0 && len(normalizedAdded) == 0 && len(normalizedRemoved) == 0 {
		return errors.New("conditions must change")
	}

	return nil
}

func normalizeConditionStateListField(values []ConditionState, field string, required bool) ([]ConditionState, bool, error) {
	if values == nil {
		if required {
			return nil, false, fmt.Errorf("%s is required", field)
		}
		return nil, false, nil
	}

	normalized, err := NormalizeConditionStates(values)
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
		hasStringFieldChange(payload.LifeStateBefore, payload.LifeStateAfter) ||
		hasClassStateFieldChange(payload.ClassStateBefore, payload.ClassStateAfter) ||
		hasSubclassStateFieldChange(payload.SubclassStateBefore, payload.SubclassStateAfter) ||
		hasBoolFieldChange(payload.ImpenetrableUsedThisShortRestBefore, payload.ImpenetrableUsedThisShortRestAfter)
}

func hasClassStateFieldChange(before, after *CharacterClassState) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return !reflect.DeepEqual(before.Normalized(), after.Normalized())
}

func hasCompanionStateFieldChange(before, after *CharacterCompanionState) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return !reflect.DeepEqual(before.Normalized(), after.Normalized())
}

func hasSubclassStateFieldChange(before, after *CharacterSubclassState) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return !reflect.DeepEqual(before.Normalized(), after.Normalized())
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

func hasRestTakeMutation(payload RestTakePayload) bool {
	if payload.GMFearBefore != payload.GMFearAfter ||
		payload.ShortRestsBefore != payload.ShortRestsAfter ||
		payload.RefreshRest ||
		payload.RefreshLongRest ||
		payload.Interrupted ||
		len(payload.CountdownUpdates) > 0 ||
		len(payload.DowntimeMoves) > 0 {
		return true
	}
	return len(payload.Participants) > 0
}

func validateRestLongTermCountdownPayload(payload CountdownUpdatePayload) error {
	if strings.TrimSpace(payload.CountdownID.String()) == "" {
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

func hasBoolFieldChange(before, after *bool) bool {
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
