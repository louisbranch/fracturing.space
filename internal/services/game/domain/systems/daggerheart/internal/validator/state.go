package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"
)

func ValidateGMFearSetPayload(raw json.RawMessage) error {
	var p payload.GMFearSetPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if p.After == nil {
		return errors.New("after is required")
	}
	if err := RequireRange(*p.After, snapstate.GMFearMin, snapstate.GMFearMax, "after"); err != nil {
		return err
	}
	return nil
}

func ValidateGMFearChangedPayload(raw json.RawMessage) error {
	var p payload.GMFearChangedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if p.Value < snapstate.GMFearMin || p.Value > snapstate.GMFearMax {
		return fmt.Errorf("value must be in range %d..%d", snapstate.GMFearMin, snapstate.GMFearMax)
	}
	return nil
}

func ValidateGMMoveApplyPayload(raw json.RawMessage) error {
	var p payload.GMMoveApplyPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if p.FearSpent <= 0 {
		return errors.New("fear_spent must be greater than zero")
	}
	return ValidateGMMoveTarget(p.Target)
}

func ValidateGMMoveAppliedPayload(raw json.RawMessage) error {
	var p payload.GMMoveAppliedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if p.FearSpent <= 0 {
		return errors.New("fear_spent must be greater than zero")
	}
	return ValidateGMMoveTarget(p.Target)
}

func ValidateGMMoveTarget(target payload.GMMoveTarget) error {
	targetType, ok := rules.NormalizeGMMoveTargetType(string(target.Type))
	if !ok {
		return errors.New("target type is unsupported")
	}
	switch targetType {
	case rules.GMMoveTargetTypeDirectMove:
		if _, ok := rules.NormalizeGMMoveKind(string(target.Kind)); !ok {
			return errors.New("kind is unsupported")
		}
		if _, ok := rules.NormalizeGMMoveShape(string(target.Shape)); !ok {
			return errors.New("shape is unsupported")
		}
		if target.Shape == rules.GMMoveShapeCustom && strings.TrimSpace(target.Description) == "" {
			return errors.New("description is required for custom shape")
		}
		if target.Shape == rules.GMMoveShapeSpotlightAdversary && strings.TrimSpace(target.AdversaryID.String()) == "" {
			return errors.New("adversary_id is required for spotlight_adversary")
		}
	case rules.GMMoveTargetTypeAdversaryFeature:
		if strings.TrimSpace(target.AdversaryID.String()) == "" {
			return errors.New("adversary_id is required")
		}
		if strings.TrimSpace(target.FeatureID) == "" {
			return errors.New("feature_id is required")
		}
	case rules.GMMoveTargetTypeEnvironmentFeature:
		if strings.TrimSpace(target.EnvironmentEntityID.String()) == "" && strings.TrimSpace(target.EnvironmentID) == "" {
			return errors.New("environment_entity_id is required")
		}
		if strings.TrimSpace(target.FeatureID) == "" {
			return errors.New("feature_id is required")
		}
	case rules.GMMoveTargetTypeAdversaryExperience:
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

func ValidateCharacterProfileReplacePayload(raw json.RawMessage) error {
	var p snapstate.CharacterProfileReplacePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.CharacterID.String(), "character_id"); err != nil {
		return err
	}
	return p.Profile.Validate()
}

func ValidateCharacterProfileReplacedPayload(raw json.RawMessage) error {
	var p snapstate.CharacterProfileReplacedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.CharacterID.String(), "character_id"); err != nil {
		return err
	}
	return p.Profile.Validate()
}

func ValidateCharacterProfileDeletePayload(raw json.RawMessage) error {
	var p snapstate.CharacterProfileDeletePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	return RequireTrimmedValue(p.CharacterID.String(), "character_id")
}

func ValidateCharacterProfileDeletedPayload(raw json.RawMessage) error {
	var p snapstate.CharacterProfileDeletedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	return RequireTrimmedValue(p.CharacterID.String(), "character_id")
}

func ValidateCharacterStatePatchPayload(raw json.RawMessage) error {
	var p payload.CharacterStatePatchPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if !HasCharacterStateChange(p) {
		return errors.New("character_state patch must change at least one field")
	}
	return nil
}

func ValidateCharacterStatePatchedPayload(raw json.RawMessage) error {
	var p payload.CharacterStatePatchedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if p.HP == nil && p.Hope == nil && p.HopeMax == nil &&
		p.Stress == nil && p.Armor == nil && p.LifeState == nil &&
		p.ClassState == nil && p.SubclassState == nil && p.ImpenetrableUsedThisShortRest == nil {
		return errors.New("character_state_patched must include at least one after field")
	}
	return nil
}

func ValidateClassFeatureApplyPayload(raw json.RawMessage) error {
	var p payload.ClassFeatureApplyPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(p.Feature) == "" {
		return errors.New("feature is required")
	}
	if len(p.Targets) == 0 {
		return errors.New("class_feature apply requires at least one target")
	}
	for _, target := range p.Targets {
		if strings.TrimSpace(target.CharacterID.String()) == "" {
			return errors.New("class_feature apply target character_id is required")
		}
		if !HasIntFieldChange(target.HPBefore, target.HPAfter) &&
			!HasIntFieldChange(target.HopeBefore, target.HopeAfter) &&
			!HasIntFieldChange(target.ArmorBefore, target.ArmorAfter) &&
			!HasClassStateFieldChange(target.ClassStateBefore, target.ClassStateAfter) {
			return errors.New("class_feature apply must change at least one field per target")
		}
	}
	return nil
}

func ValidateSubclassFeatureApplyPayload(raw json.RawMessage) error {
	var p payload.SubclassFeatureApplyPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(p.Feature) == "" {
		return errors.New("feature is required")
	}
	if len(p.Targets) == 0 && len(p.CharacterConditionTargets) == 0 && len(p.AdversaryConditionTargets) == 0 {
		return errors.New("subclass_feature apply requires at least one consequence")
	}
	hasMutation := len(p.CharacterConditionTargets) > 0 || len(p.AdversaryConditionTargets) > 0
	for _, target := range p.Targets {
		if strings.TrimSpace(target.CharacterID.String()) == "" {
			return errors.New("subclass_feature apply target character_id is required")
		}
		if !HasIntFieldChange(target.HPBefore, target.HPAfter) &&
			!HasIntFieldChange(target.HopeBefore, target.HopeAfter) &&
			!HasIntFieldChange(target.StressBefore, target.StressAfter) &&
			!HasIntFieldChange(target.ArmorBefore, target.ArmorAfter) &&
			!HasClassStateFieldChange(target.ClassStateBefore, target.ClassStateAfter) &&
			!HasSubclassStateFieldChange(target.SubclassStateBefore, target.SubclassStateAfter) {
			continue
		}
		hasMutation = true
	}
	if !hasMutation {
		return errors.New("subclass_feature apply must change at least one field")
	}
	return nil
}

func ValidateBeastformTransformPayload(raw json.RawMessage) error {
	var p payload.BeastformTransformPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.BeastformID) == "" {
		return errors.New("beastform_id is required")
	}
	if !HasIntFieldChange(p.HopeBefore, p.HopeAfter) &&
		!HasIntFieldChange(p.StressBefore, p.StressAfter) &&
		!HasClassStateFieldChange(p.ClassStateBefore, p.ClassStateAfter) {
		return errors.New("beastform transform must change at least one field")
	}
	return nil
}

func ValidateBeastformDropPayload(raw json.RawMessage) error {
	var p payload.BeastformDropPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.BeastformID) == "" {
		return errors.New("beastform_id is required")
	}
	if !HasClassStateFieldChange(p.ClassStateBefore, p.ClassStateAfter) {
		return errors.New("beastform drop must change class state")
	}
	return nil
}

func ValidateBeastformTransformedPayload(raw json.RawMessage) error {
	var p payload.BeastformTransformedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.BeastformID) == "" {
		return errors.New("beastform_id is required")
	}
	if snapstate.NormalizedActiveBeastformPtr(p.ActiveBeastform) == nil {
		return errors.New("active_beastform is required")
	}
	return nil
}

func ValidateBeastformDroppedPayload(raw json.RawMessage) error {
	var p payload.BeastformDroppedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.BeastformID) == "" {
		return errors.New("beastform_id is required")
	}
	return nil
}

func ValidateCompanionExperienceBeginPayload(raw json.RawMessage) error {
	var p payload.CompanionExperienceBeginPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.ExperienceID) == "" {
		return errors.New("experience_id is required")
	}
	if !HasCompanionStateFieldChange(p.CompanionStateBefore, p.CompanionStateAfter) {
		return errors.New("companion begin must change companion state")
	}
	return nil
}

func ValidateCompanionReturnPayload(raw json.RawMessage) error {
	var p payload.CompanionReturnPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.Resolution) == "" {
		return errors.New("resolution is required")
	}
	if !HasIntFieldChange(p.StressBefore, p.StressAfter) &&
		!HasCompanionStateFieldChange(p.CompanionStateBefore, p.CompanionStateAfter) {
		return errors.New("companion return must change at least one field")
	}
	return nil
}

func ValidateCompanionExperienceBegunPayload(raw json.RawMessage) error {
	var p payload.CompanionExperienceBegunPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.ExperienceID) == "" {
		return errors.New("experience_id is required")
	}
	if snapstate.NormalizedCompanionStatePtr(p.CompanionState) == nil {
		return errors.New("companion_state is required")
	}
	return nil
}

func ValidateCompanionReturnedPayload(raw json.RawMessage) error {
	var p payload.CompanionReturnedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.Resolution) == "" {
		return errors.New("resolution is required")
	}
	if snapstate.NormalizedCompanionStatePtr(p.CompanionState) == nil {
		return errors.New("companion_state is required")
	}
	return nil
}

func ValidateHopeSpendPayload(raw json.RawMessage) error {
	var p payload.HopeSpendPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if p.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if p.Before == p.After {
		return errors.New("before and after must differ")
	}
	if Abs(p.Before-p.After) != p.Amount {
		return errors.New("amount must match before and after delta")
	}
	return nil
}

func ValidateStressSpendPayload(raw json.RawMessage) error {
	var p payload.StressSpendPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if p.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if p.Before == p.After {
		return errors.New("before and after must differ")
	}
	if Abs(p.Before-p.After) != p.Amount {
		return errors.New("amount must match before and after delta")
	}
	return nil
}

func ValidateConditionChangePayload(raw json.RawMessage) error {
	var p payload.ConditionChangePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	return ValidateConditionSetPayload(
		p.ConditionsBefore,
		p.ConditionsAfter,
		p.Added,
		p.Removed,
	)
}

func ValidateConditionChangedPayload(raw json.RawMessage) error {
	var p payload.ConditionChangedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if p.Conditions == nil {
		return errors.New("conditions_after is required")
	}
	if _, err := rules.NormalizeConditionStates(p.Conditions); err != nil {
		return fmt.Errorf("conditions_after: %w", err)
	}
	return nil
}

func ValidateLoadoutSwapPayload(raw json.RawMessage) error {
	var p payload.LoadoutSwapPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.CharacterID.String(), "character_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.CardID, "card_id"); err != nil {
		return err
	}
	return nil
}

func ValidateLoadoutSwappedPayload(raw json.RawMessage) error {
	var p payload.LoadoutSwappedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.CharacterID.String(), "character_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.CardID, "card_id"); err != nil {
		return err
	}
	return nil
}

func ValidateRestTakePayload(raw json.RawMessage) error {
	var p payload.RestTakePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.RestType, "rest_type"); err != nil {
		return err
	}
	if len(p.Participants) == 0 {
		return errors.New("participants are required")
	}
	for _, participantID := range p.Participants {
		if err := RequireTrimmedValue(participantID.String(), "participants.character_id"); err != nil {
			return err
		}
	}
	for _, update := range p.CountdownUpdates {
		if err := ValidateRestLongTermCountdownPayload(update); err != nil {
			return err
		}
	}
	for _, move := range p.DowntimeMoves {
		if err := ValidateDowntimeMoveAppliedPayloadFields(move); err != nil {
			return err
		}
	}
	if !HasRestTakeMutation(p) {
		return errors.New("rest.take must record at least one durable outcome")
	}
	return nil
}

func ValidateRestTakenPayload(raw json.RawMessage) error {
	var p payload.RestTakenPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.RestType, "rest_type"); err != nil {
		return err
	}
	if p.GMFear < snapstate.GMFearMin || p.GMFear > snapstate.GMFearMax {
		return fmt.Errorf("gm_fear_after must be in range %d..%d", snapstate.GMFearMin, snapstate.GMFearMax)
	}
	if len(p.Participants) == 0 {
		return errors.New("participants are required")
	}
	return nil
}

func ValidateCountdownCreatePayload(raw json.RawMessage) error {
	var p payload.CountdownCreatePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.CountdownID.String(), "countdown_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.Name, "name"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.Kind, "kind"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.Direction, "direction"); err != nil {
		return err
	}
	if err := RequirePositive(p.Max, "max"); err != nil {
		return err
	}
	if p.Current < 0 || p.Current > p.Max {
		return fmt.Errorf("current must be in range 0..%d", p.Max)
	}
	variant := strings.TrimSpace(p.Variant)
	if variant == "" {
		variant = "standard"
	}
	switch variant {
	case "standard", "dynamic", "linked":
		// valid
	default:
		return fmt.Errorf("unknown countdown variant %q; must be standard, dynamic, or linked", variant)
	}
	if variant == "dynamic" && strings.TrimSpace(p.TriggerEventType) == "" {
		return errors.New("trigger_event_type is required for dynamic countdowns")
	}
	if variant == "linked" && strings.TrimSpace(p.LinkedCountdownID.String()) == "" {
		return errors.New("linked_countdown_id is required for linked countdowns")
	}
	return nil
}

func ValidateCountdownCreatedPayload(raw json.RawMessage) error {
	return ValidateCountdownCreatePayload(raw)
}

func ValidateCountdownUpdatePayload(raw json.RawMessage) error {
	var p payload.CountdownUpdatePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CountdownID.String()) == "" {
		return errors.New("countdown_id is required")
	}
	if p.Before == p.After && p.Delta == 0 {
		return errors.New("countdown update must change value")
	}
	return nil
}

func ValidateCountdownUpdatedPayload(raw json.RawMessage) error {
	var p payload.CountdownUpdatedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CountdownID.String()) == "" {
		return errors.New("countdown_id is required")
	}
	return nil
}

func ValidateCountdownDeletePayload(raw json.RawMessage) error {
	var p payload.CountdownDeletePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CountdownID.String()) == "" {
		return errors.New("countdown_id is required")
	}
	return nil
}

func ValidateCountdownDeletedPayload(raw json.RawMessage) error {
	return ValidateCountdownDeletePayload(raw)
}

func ValidateAdversaryConditionChangePayload(raw json.RawMessage) error {
	var p payload.AdversaryConditionChangePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.AdversaryID.String()) == "" {
		return errors.New("adversary_id is required")
	}
	return ValidateConditionSetPayload(
		p.ConditionsBefore,
		p.ConditionsAfter,
		p.Added,
		p.Removed,
	)
}

func ValidateAdversaryConditionChangedPayload(raw json.RawMessage) error {
	var p payload.AdversaryConditionChangedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.AdversaryID.String()) == "" {
		return errors.New("adversary_id is required")
	}
	if p.Conditions == nil {
		return errors.New("conditions_after is required")
	}
	if _, err := rules.NormalizeConditionStates(p.Conditions); err != nil {
		return fmt.Errorf("conditions_after: %w", err)
	}
	return nil
}

func ValidateConditionSetPayload(before, after, added, removed []rules.ConditionState) error {
	normalizedAfter, _, err := NormalizeConditionStateListField(after, "conditions_after", true)
	if err != nil {
		return err
	}

	normalizedBefore, hasBefore, err := NormalizeConditionStateListField(before, "conditions_before", false)
	if err != nil {
		return err
	}
	normalizedAdded, hasAdded, err := NormalizeConditionStateListField(added, "added", false)
	if err != nil {
		return err
	}
	normalizedRemoved, hasRemoved, err := NormalizeConditionStateListField(removed, "removed", false)
	if err != nil {
		return err
	}

	expectedAdded := normalizedAfter
	expectedRemoved := []rules.ConditionState{}
	if hasBefore {
		expectedAdded, expectedRemoved = rules.DiffConditionStates(normalizedBefore, normalizedAfter)
	}

	if !hasBefore && hasRemoved && len(normalizedRemoved) > 0 {
		return errors.New("conditions_before is required when removed are provided")
	}

	if hasAdded {
		if !rules.ConditionStatesEqual(normalizedAdded, expectedAdded) {
			if hasBefore {
				return errors.New("added must match conditions_before and conditions_after diff")
			}
			return errors.New("added must match conditions_after when conditions_before is omitted")
		}
	}

	if hasRemoved && !rules.ConditionStatesEqual(normalizedRemoved, expectedRemoved) {
		if hasBefore {
			return errors.New("removed must match conditions_before and conditions_after diff")
		}
		return errors.New("removed must be empty when conditions_before is omitted")
	}

	if hasBefore {
		if rules.ConditionStatesEqual(normalizedBefore, normalizedAfter) &&
			len(normalizedAdded) == 0 && len(normalizedRemoved) == 0 {
			return errors.New("conditions must change")
		}
	} else if len(normalizedAfter) == 0 && len(normalizedAdded) == 0 && len(normalizedRemoved) == 0 {
		return errors.New("conditions must change")
	}

	return nil
}

func NormalizeConditionStateListField(values []rules.ConditionState, field string, required bool) ([]rules.ConditionState, bool, error) {
	if values == nil {
		if required {
			return nil, false, fmt.Errorf("%s is required", field)
		}
		return nil, false, nil
	}

	normalized, err := rules.NormalizeConditionStates(values)
	if err != nil {
		return nil, true, fmt.Errorf("%s: %w", field, err)
	}
	return normalized, true, nil
}

func HasCharacterStateChange(p payload.CharacterStatePatchPayload) bool {
	return HasIntFieldChange(p.HPBefore, p.HPAfter) ||
		HasIntFieldChange(p.HopeBefore, p.HopeAfter) ||
		HasIntFieldChange(p.HopeMaxBefore, p.HopeMaxAfter) ||
		HasIntFieldChange(p.StressBefore, p.StressAfter) ||
		HasIntFieldChange(p.ArmorBefore, p.ArmorAfter) ||
		HasStringFieldChange(p.LifeStateBefore, p.LifeStateAfter) ||
		HasClassStateFieldChange(p.ClassStateBefore, p.ClassStateAfter) ||
		HasSubclassStateFieldChange(p.SubclassStateBefore, p.SubclassStateAfter) ||
		HasBoolFieldChange(p.ImpenetrableUsedThisShortRestBefore, p.ImpenetrableUsedThisShortRestAfter)
}

func HasClassStateFieldChange(before, after *snapstate.CharacterClassState) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return !reflect.DeepEqual(before.Normalized(), after.Normalized())
}

func HasCompanionStateFieldChange(before, after *snapstate.CharacterCompanionState) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return !reflect.DeepEqual(before.Normalized(), after.Normalized())
}

func HasSubclassStateFieldChange(before, after *snapstate.CharacterSubclassState) bool {
	if before == nil && after == nil {
		return false
	}
	if before == nil || after == nil {
		return true
	}
	return !reflect.DeepEqual(before.Normalized(), after.Normalized())
}

func HasConditionListMutation(before, after []string) bool {
	beforeNormalized, err := rules.NormalizeConditions(before)
	if err != nil {
		return true
	}
	afterNormalized, err := rules.NormalizeConditions(after)
	if err != nil {
		return true
	}
	return !rules.ConditionsEqual(beforeNormalized, afterNormalized)
}

func HasRestTakeMutation(p payload.RestTakePayload) bool {
	if p.GMFearBefore != p.GMFearAfter ||
		p.ShortRestsBefore != p.ShortRestsAfter ||
		p.RefreshRest ||
		p.RefreshLongRest ||
		p.Interrupted ||
		len(p.CountdownUpdates) > 0 ||
		len(p.DowntimeMoves) > 0 {
		return true
	}
	return len(p.Participants) > 0
}

func ValidateRestLongTermCountdownPayload(p payload.CountdownUpdatePayload) error {
	if strings.TrimSpace(p.CountdownID.String()) == "" {
		return errors.New("long_term_countdown.countdown_id is required")
	}
	if p.Before == p.After && p.Delta == 0 {
		return errors.New("long_term_countdown must change value")
	}
	return nil
}

func HasIntFieldChange(before, after *int) bool {
	if after == nil {
		return false
	}
	if before == nil {
		return true
	}
	return *before != *after
}

func HasStringFieldChange(before, after *string) bool {
	if after == nil {
		return false
	}
	if before == nil {
		return true
	}
	return *before != *after
}

func HasBoolFieldChange(before, after *bool) bool {
	if after == nil {
		return false
	}
	if before == nil {
		return true
	}
	return *before != *after
}

func Abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
