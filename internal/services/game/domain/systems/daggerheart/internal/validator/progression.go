package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/rules"
)

func ValidateAdversaryCreatePayload(raw json.RawMessage) error {
	var p payload.AdversaryCreatePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.AdversaryID.String(), "adversary_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.AdversaryEntryID, "adversary_entry_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.Name, "name"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.SessionID.String(), "session_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.SceneID.String(), "scene_id"); err != nil {
		return err
	}
	return nil
}

func ValidateAdversaryCreatedPayload(raw json.RawMessage) error {
	return ValidateAdversaryCreatePayload(raw)
}

func ValidateAdversaryUpdatePayload(raw json.RawMessage) error {
	var p payload.AdversaryUpdatePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.AdversaryID.String(), "adversary_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.AdversaryEntryID, "adversary_entry_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.Name, "name"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.SessionID.String(), "session_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.SceneID.String(), "scene_id"); err != nil {
		return err
	}
	return nil
}

func ValidateAdversaryUpdatedPayload(raw json.RawMessage) error {
	return ValidateAdversaryUpdatePayload(raw)
}

func ValidateAdversaryFeatureApplyPayload(raw json.RawMessage) error {
	var p payload.AdversaryFeatureApplyPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.ActorAdversaryID.String(), "actor_adversary_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.AdversaryID.String(), "adversary_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.FeatureID, "feature_id"); err != nil {
		return err
	}
	hasMutation := HasIntFieldChange(p.StressBefore, p.StressAfter) ||
		!EqualAdversaryFeatureStates(p.FeatureStatesBefore, p.FeatureStatesAfter) ||
		!EqualAdversaryPendingExperience(p.PendingExperienceBefore, p.PendingExperienceAfter)
	if !hasMutation {
		return errors.New("adversary_feature apply must change at least one field")
	}
	return nil
}

func ValidateAdversaryDeletePayload(raw json.RawMessage) error {
	var p payload.AdversaryDeletePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.AdversaryID.String()) == "" {
		return errors.New("adversary_id is required")
	}
	return nil
}

func ValidateAdversaryDeletedPayload(raw json.RawMessage) error {
	return ValidateAdversaryDeletePayload(raw)
}

func EqualAdversaryFeatureStates(before, after []rules.AdversaryFeatureState) bool {
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

func EqualAdversaryPendingExperience(before, after *rules.AdversaryPendingExperience) bool {
	if before == nil || after == nil {
		return before == after
	}
	return *before == *after
}

func ValidateEnvironmentEntityCreatePayload(raw json.RawMessage) error {
	var p payload.EnvironmentEntityCreatePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.EnvironmentEntityID.String(), "environment_entity_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.EnvironmentID, "environment_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.Name, "name"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.Type, "type"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.SessionID.String(), "session_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.SceneID.String(), "scene_id"); err != nil {
		return err
	}
	if p.Tier < 0 {
		return errors.New("tier must be non-negative")
	}
	if p.Difficulty <= 0 {
		return errors.New("difficulty must be positive")
	}
	return nil
}

func ValidateEnvironmentEntityCreatedPayload(raw json.RawMessage) error {
	return ValidateEnvironmentEntityCreatePayload(raw)
}

func ValidateEnvironmentEntityUpdatePayload(raw json.RawMessage) error {
	var p payload.EnvironmentEntityUpdatePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.EnvironmentEntityID.String(), "environment_entity_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.EnvironmentID, "environment_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.Name, "name"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.Type, "type"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.SessionID.String(), "session_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.SceneID.String(), "scene_id"); err != nil {
		return err
	}
	if p.Tier < 0 {
		return errors.New("tier must be non-negative")
	}
	if p.Difficulty <= 0 {
		return errors.New("difficulty must be positive")
	}
	return nil
}

func ValidateEnvironmentEntityUpdatedPayload(raw json.RawMessage) error {
	return ValidateEnvironmentEntityUpdatePayload(raw)
}

func ValidateEnvironmentEntityDeletePayload(raw json.RawMessage) error {
	var p payload.EnvironmentEntityDeletePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.EnvironmentEntityID.String()) == "" {
		return errors.New("environment_entity_id is required")
	}
	return nil
}

func ValidateEnvironmentEntityDeletedPayload(raw json.RawMessage) error {
	return ValidateEnvironmentEntityDeletePayload(raw)
}

func ValidateLevelUpApplyPayload(raw json.RawMessage) error {
	var p payload.LevelUpApplyPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if p.LevelBefore < 1 || p.LevelBefore > 10 {
		return fmt.Errorf("level_before must be in range 1..10")
	}
	if p.LevelAfter < 1 || p.LevelAfter > 10 {
		return fmt.Errorf("level_after must be in range 1..10")
	}
	if p.LevelAfter != p.LevelBefore+1 {
		return fmt.Errorf("level_after must be level_before + 1")
	}
	if len(p.Advancements) == 0 {
		return errors.New("advancements is required")
	}
	for _, reward := range p.Rewards {
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

func ValidateLevelUpAppliedPayload(raw json.RawMessage) error {
	var p payload.LevelUpAppliedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if p.Level < 1 || p.Level > 10 {
		return fmt.Errorf("level_after must be in range 1..10")
	}
	if len(p.Advancements) == 0 {
		return errors.New("advancements is required")
	}
	return nil
}

func ValidateGoldUpdatePayload(raw json.RawMessage) error {
	var p payload.GoldUpdatePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if p.HandfulsAfter < 0 || p.HandfulsAfter > 9 {
		return errors.New("handfuls_after must be in range 0..9")
	}
	if p.BagsAfter < 0 || p.BagsAfter > 9 {
		return errors.New("bags_after must be in range 0..9")
	}
	if p.ChestsAfter < 0 || p.ChestsAfter > 1 {
		return errors.New("chests_after must be in range 0..1")
	}
	if p.HandfulsBefore == p.HandfulsAfter &&
		p.BagsBefore == p.BagsAfter &&
		p.ChestsBefore == p.ChestsAfter {
		return errors.New("gold update must change at least one denomination")
	}
	return nil
}

func ValidateGoldUpdatedPayload(raw json.RawMessage) error {
	var p payload.GoldUpdatedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if p.Handfuls < 0 || p.Handfuls > 9 {
		return errors.New("handfuls_after must be in range 0..9")
	}
	if p.Bags < 0 || p.Bags > 9 {
		return errors.New("bags_after must be in range 0..9")
	}
	if p.Chests < 0 || p.Chests > 1 {
		return errors.New("chests_after must be in range 0..1")
	}
	return nil
}

func ValidateDomainCardAcquirePayload(raw json.RawMessage) error {
	var p payload.DomainCardAcquirePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.CardID) == "" {
		return errors.New("card_id is required")
	}
	if p.CardLevel < 1 {
		return errors.New("card_level must be at least 1")
	}
	dest := strings.TrimSpace(p.Destination)
	if dest != "vault" && dest != "loadout" {
		return errors.New("destination must be vault or loadout")
	}
	return nil
}

func ValidateDomainCardAcquiredPayload(raw json.RawMessage) error {
	return ValidateDomainCardAcquirePayload(raw)
}

func ValidateEquipmentSwapPayload(raw json.RawMessage) error {
	var p payload.EquipmentSwapPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.ItemID) == "" {
		return errors.New("item_id is required")
	}
	itemType := strings.TrimSpace(p.ItemType)
	if itemType != "weapon" && itemType != "armor" {
		return errors.New("item_type must be weapon or armor")
	}
	from := strings.TrimSpace(p.From)
	to := strings.TrimSpace(p.To)
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

func ValidateEquipmentSwappedPayload(raw json.RawMessage) error {
	return ValidateEquipmentSwapPayload(raw)
}

func ValidateConsumableUsePayload(raw json.RawMessage) error {
	var p payload.ConsumableUsePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.ConsumableID) == "" {
		return errors.New("consumable_id is required")
	}
	if p.QuantityBefore <= 0 {
		return errors.New("quantity_before must be positive")
	}
	if p.QuantityAfter != p.QuantityBefore-1 {
		return errors.New("quantity_after must be quantity_before - 1")
	}
	return nil
}

func ValidateConsumableUsedPayload(raw json.RawMessage) error {
	var p payload.ConsumableUsedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.ConsumableID) == "" {
		return errors.New("consumable_id is required")
	}
	return nil
}

func ValidateConsumableAcquirePayload(raw json.RawMessage) error {
	var p payload.ConsumableAcquirePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.ConsumableID) == "" {
		return errors.New("consumable_id is required")
	}
	if p.QuantityAfter < 1 || p.QuantityAfter > 5 {
		return errors.New("quantity_after must be in range 1..5")
	}
	if p.QuantityAfter != p.QuantityBefore+1 {
		return errors.New("quantity_after must be quantity_before + 1")
	}
	return nil
}

func ValidateConsumableAcquiredPayload(raw json.RawMessage) error {
	var p payload.ConsumableAcquiredPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(p.ConsumableID) == "" {
		return errors.New("consumable_id is required")
	}
	if p.Quantity < 1 || p.Quantity > 5 {
		return errors.New("quantity_after must be in range 1..5")
	}
	return nil
}
