package daggerheart

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func validateAdversaryCreatePayload(raw json.RawMessage) error {
	var payload AdversaryCreatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.AdversaryID.String(), "adversary_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.AdversaryEntryID, "adversary_entry_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.Name, "name"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.SessionID.String(), "session_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.SceneID.String(), "scene_id"); err != nil {
		return err
	}
	return nil
}

func validateAdversaryCreatedPayload(raw json.RawMessage) error {
	return validateAdversaryCreatePayload(raw)
}

func validateAdversaryUpdatePayload(raw json.RawMessage) error {
	var payload AdversaryUpdatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.AdversaryID.String(), "adversary_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.AdversaryEntryID, "adversary_entry_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.Name, "name"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.SessionID.String(), "session_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.SceneID.String(), "scene_id"); err != nil {
		return err
	}
	return nil
}

func validateAdversaryUpdatedPayload(raw json.RawMessage) error {
	return validateAdversaryUpdatePayload(raw)
}

func validateAdversaryFeatureApplyPayload(raw json.RawMessage) error {
	var payload AdversaryFeatureApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.ActorAdversaryID.String(), "actor_adversary_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.AdversaryID.String(), "adversary_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.FeatureID, "feature_id"); err != nil {
		return err
	}
	hasMutation := hasIntFieldChange(payload.StressBefore, payload.StressAfter) ||
		!equalAdversaryFeatureStates(payload.FeatureStatesBefore, payload.FeatureStatesAfter) ||
		!equalAdversaryPendingExperience(payload.PendingExperienceBefore, payload.PendingExperienceAfter)
	if !hasMutation {
		return errors.New("adversary_feature apply must change at least one field")
	}
	return nil
}

func validateAdversaryDeletePayload(raw json.RawMessage) error {
	var payload AdversaryDeletePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID.String()) == "" {
		return errors.New("adversary_id is required")
	}
	return nil
}

func validateAdversaryDeletedPayload(raw json.RawMessage) error {
	return validateAdversaryDeletePayload(raw)
}

func equalAdversaryFeatureStates(before, after []AdversaryFeatureState) bool {
	if len(before) != len(after) {
		return false
	}
	for i := range before {
		if before[i] != after[i] {
			return false
		}
	}
	return true
}

func equalAdversaryPendingExperience(before, after *AdversaryPendingExperience) bool {
	if before == nil || after == nil {
		return before == after
	}
	return *before == *after
}

func validateEnvironmentEntityCreatePayload(raw json.RawMessage) error {
	var payload EnvironmentEntityCreatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.EnvironmentEntityID.String(), "environment_entity_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.EnvironmentID, "environment_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.Name, "name"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.Type, "type"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.SessionID.String(), "session_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.SceneID.String(), "scene_id"); err != nil {
		return err
	}
	if payload.Tier < 0 {
		return errors.New("tier must be non-negative")
	}
	if payload.Difficulty <= 0 {
		return errors.New("difficulty must be positive")
	}
	return nil
}

func validateEnvironmentEntityCreatedPayload(raw json.RawMessage) error {
	return validateEnvironmentEntityCreatePayload(raw)
}

func validateEnvironmentEntityUpdatePayload(raw json.RawMessage) error {
	var payload EnvironmentEntityUpdatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.EnvironmentEntityID.String(), "environment_entity_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.EnvironmentID, "environment_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.Name, "name"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.Type, "type"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.SessionID.String(), "session_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.SceneID.String(), "scene_id"); err != nil {
		return err
	}
	if payload.Tier < 0 {
		return errors.New("tier must be non-negative")
	}
	if payload.Difficulty <= 0 {
		return errors.New("difficulty must be positive")
	}
	return nil
}

func validateEnvironmentEntityUpdatedPayload(raw json.RawMessage) error {
	return validateEnvironmentEntityUpdatePayload(raw)
}

func validateEnvironmentEntityDeletePayload(raw json.RawMessage) error {
	var payload EnvironmentEntityDeletePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.EnvironmentEntityID.String()) == "" {
		return errors.New("environment_entity_id is required")
	}
	return nil
}

func validateEnvironmentEntityDeletedPayload(raw json.RawMessage) error {
	return validateEnvironmentEntityDeletePayload(raw)
}

func validateLevelUpApplyPayload(raw json.RawMessage) error {
	var payload LevelUpApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if payload.LevelBefore < 1 || payload.LevelBefore > 10 {
		return fmt.Errorf("level_before must be in range 1..10")
	}
	if payload.LevelAfter < 1 || payload.LevelAfter > 10 {
		return fmt.Errorf("level_after must be in range 1..10")
	}
	if payload.LevelAfter != payload.LevelBefore+1 {
		return fmt.Errorf("level_after must be level_before + 1")
	}
	if len(payload.Advancements) == 0 {
		return errors.New("advancements is required")
	}
	for _, reward := range payload.Rewards {
		switch strings.TrimSpace(reward.Type) {
		case "domain_card":
			if strings.TrimSpace(reward.DomainCardID) == "" {
				return errors.New("reward domain_card_id is required")
			}
			if reward.DomainCardLevel < 1 {
				return errors.New("reward domain_card_level must be at least 1")
			}
		case "companion_bonus_choices":
			if reward.CompanionBonusChoices <= 0 {
				return errors.New("reward companion_bonus_choices must be positive")
			}
		case "":
			return errors.New("reward type is required")
		default:
			return fmt.Errorf("reward type %q is unsupported", reward.Type)
		}
	}
	return nil
}

func validateLevelUpAppliedPayload(raw json.RawMessage) error {
	var payload LevelUpAppliedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if payload.Level < 1 || payload.Level > 10 {
		return fmt.Errorf("level_after must be in range 1..10")
	}
	if len(payload.Advancements) == 0 {
		return errors.New("advancements is required")
	}
	return nil
}

func validateGoldUpdatePayload(raw json.RawMessage) error {
	var payload GoldUpdatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if payload.HandfulsAfter < 0 || payload.HandfulsAfter > 9 {
		return errors.New("handfuls_after must be in range 0..9")
	}
	if payload.BagsAfter < 0 || payload.BagsAfter > 9 {
		return errors.New("bags_after must be in range 0..9")
	}
	if payload.ChestsAfter < 0 || payload.ChestsAfter > 1 {
		return errors.New("chests_after must be in range 0..1")
	}
	if payload.HandfulsBefore == payload.HandfulsAfter &&
		payload.BagsBefore == payload.BagsAfter &&
		payload.ChestsBefore == payload.ChestsAfter {
		return errors.New("gold update must change at least one denomination")
	}
	return nil
}

func validateGoldUpdatedPayload(raw json.RawMessage) error {
	var payload GoldUpdatedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if payload.Handfuls < 0 || payload.Handfuls > 9 {
		return errors.New("handfuls_after must be in range 0..9")
	}
	if payload.Bags < 0 || payload.Bags > 9 {
		return errors.New("bags_after must be in range 0..9")
	}
	if payload.Chests < 0 || payload.Chests > 1 {
		return errors.New("chests_after must be in range 0..1")
	}
	return nil
}

func validateDomainCardAcquirePayload(raw json.RawMessage) error {
	var payload DomainCardAcquirePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.CardID) == "" {
		return errors.New("card_id is required")
	}
	if payload.CardLevel < 1 {
		return errors.New("card_level must be at least 1")
	}
	dest := strings.TrimSpace(payload.Destination)
	if dest != "vault" && dest != "loadout" {
		return errors.New("destination must be vault or loadout")
	}
	return nil
}

func validateDomainCardAcquiredPayload(raw json.RawMessage) error {
	return validateDomainCardAcquirePayload(raw)
}

func validateEquipmentSwapPayload(raw json.RawMessage) error {
	var payload EquipmentSwapPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.ItemID) == "" {
		return errors.New("item_id is required")
	}
	itemType := strings.TrimSpace(payload.ItemType)
	if itemType != "weapon" && itemType != "armor" {
		return errors.New("item_type must be weapon or armor")
	}
	from := strings.TrimSpace(payload.From)
	to := strings.TrimSpace(payload.To)
	validSlot := func(s string) bool {
		return s == "active" || s == "inventory" || s == "none"
	}
	if !validSlot(from) || !validSlot(to) {
		return errors.New("from and to must be active, inventory, or none")
	}
	if from == to {
		return errors.New("from and to must differ")
	}
	return nil
}

func validateEquipmentSwappedPayload(raw json.RawMessage) error {
	return validateEquipmentSwapPayload(raw)
}

func validateConsumableUsePayload(raw json.RawMessage) error {
	var payload ConsumableUsePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.ConsumableID) == "" {
		return errors.New("consumable_id is required")
	}
	if payload.QuantityBefore <= 0 {
		return errors.New("quantity_before must be positive")
	}
	if payload.QuantityAfter != payload.QuantityBefore-1 {
		return errors.New("quantity_after must be quantity_before - 1")
	}
	return nil
}

func validateConsumableUsedPayload(raw json.RawMessage) error {
	var payload ConsumableUsedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.ConsumableID) == "" {
		return errors.New("consumable_id is required")
	}
	return nil
}

func validateConsumableAcquirePayload(raw json.RawMessage) error {
	var payload ConsumableAcquirePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.ConsumableID) == "" {
		return errors.New("consumable_id is required")
	}
	if payload.QuantityAfter < 1 || payload.QuantityAfter > 5 {
		return errors.New("quantity_after must be in range 1..5")
	}
	if payload.QuantityAfter != payload.QuantityBefore+1 {
		return errors.New("quantity_after must be quantity_before + 1")
	}
	return nil
}

func validateConsumableAcquiredPayload(raw json.RawMessage) error {
	var payload ConsumableAcquiredPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.ConsumableID) == "" {
		return errors.New("consumable_id is required")
	}
	if payload.Quantity < 1 || payload.Quantity > 5 {
		return errors.New("quantity_after must be in range 1..5")
	}
	return nil
}
