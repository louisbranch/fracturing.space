package daggerheart

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func validateDamageApplyPayload(raw json.RawMessage) error {
	var payload DamageApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if !hasDamagePatchMutation(payload.HpBefore, payload.HpAfter, payload.ArmorBefore, payload.ArmorAfter) {
		return errors.New("damage apply must change hp or armor")
	}
	if err := validateDamageAdapterInvariants(payload); err != nil {
		return err
	}
	return nil
}

func validateDamageAppliedPayload(raw json.RawMessage) error {
	var payload DamageAppliedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if payload.Hp == nil && payload.Armor == nil {
		return errors.New("damage applied must include hp or armor")
	}
	return validateDamageAppliedInvariants(payload)
}

func validateDamageAppliedInvariants(payload DamageAppliedPayload) error {
	if payload.ArmorSpent < 0 || payload.ArmorSpent > ArmorMaxCap {
		return fmt.Errorf("armor_spent must be in range 0..%d", ArmorMaxCap)
	}
	if payload.Marks < 0 || payload.Marks > MaxDamageMarks {
		return fmt.Errorf("marks must be in range 0..%d", MaxDamageMarks)
	}
	if payload.RollSeq != nil && *payload.RollSeq == 0 {
		return errors.New("roll_seq must be positive")
	}
	if severity := strings.TrimSpace(payload.Severity); severity != "" {
		switch severity {
		case "none", "minor", "major", "severe", "massive":
			// allowed
		default:
			return errors.New("severity must be one of none, minor, major, severe, massive")
		}
	}
	for _, id := range payload.SourceCharacterIDs {
		if strings.TrimSpace(id.String()) == "" {
			return errors.New("source_character_ids must not contain empty values")
		}
	}
	return nil
}

func validateMultiTargetDamageApplyPayload(raw json.RawMessage) error {
	var payload MultiTargetDamageApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if len(payload.Targets) == 0 {
		return errors.New("targets is required and must not be empty")
	}
	for i, t := range payload.Targets {
		if strings.TrimSpace(t.CharacterID.String()) == "" {
			return fmt.Errorf("targets[%d]: character_id is required", i)
		}
		if !hasDamagePatchMutation(t.HpBefore, t.HpAfter, t.ArmorBefore, t.ArmorAfter) {
			return fmt.Errorf("targets[%d]: damage apply must change hp or armor", i)
		}
		if err := validateDamageAdapterInvariants(t); err != nil {
			return fmt.Errorf("targets[%d]: %w", i, err)
		}
	}
	return nil
}

func validateAdversaryDamageApplyPayload(raw json.RawMessage) error {
	var payload AdversaryDamageApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID.String()) == "" {
		return errors.New("adversary_id is required")
	}
	if !hasDamagePatchMutation(payload.HpBefore, payload.HpAfter, payload.ArmorBefore, payload.ArmorAfter) {
		return errors.New("damage apply must change hp or armor")
	}
	return nil
}

func validateAdversaryDamageAppliedPayload(raw json.RawMessage) error {
	var payload AdversaryDamageAppliedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.AdversaryID.String()) == "" {
		return errors.New("adversary_id is required")
	}
	if payload.Hp == nil && payload.Armor == nil {
		return errors.New("damage applied must include hp or armor")
	}
	return nil
}

func validateDamageAdapterInvariants(payload DamageApplyPayload) error {
	if payload.ArmorSpent < 0 || payload.ArmorSpent > ArmorMaxCap {
		return fmt.Errorf("armor_spent must be in range 0..%d", ArmorMaxCap)
	}
	if payload.Marks < 0 || payload.Marks > MaxDamageMarks {
		return fmt.Errorf("marks must be in range 0..%d", MaxDamageMarks)
	}
	if payload.RollSeq != nil && *payload.RollSeq == 0 {
		return errors.New("roll_seq must be positive")
	}
	if severity := strings.TrimSpace(payload.Severity); severity != "" {
		switch severity {
		case "none", "minor", "major", "severe", "massive":
			// allowed
		default:
			return errors.New("severity must be one of none, minor, major, severe, massive")
		}
	}
	for _, id := range payload.SourceCharacterIDs {
		if strings.TrimSpace(id.String()) == "" {
			return errors.New("source_character_ids must not contain empty values")
		}
	}
	return nil
}

func validateDowntimeMoveApplyPayload(raw json.RawMessage) error {
	var payload DowntimeMoveApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.Move) == "" {
		return errors.New("move is required")
	}
	if !hasIntFieldChange(payload.HopeBefore, payload.HopeAfter) &&
		!hasIntFieldChange(payload.StressBefore, payload.StressAfter) &&
		!hasIntFieldChange(payload.ArmorBefore, payload.ArmorAfter) {
		return errors.New("downtime_move must change at least one state field")
	}
	return nil
}

func validateDowntimeMoveAppliedPayload(raw json.RawMessage) error {
	var payload DowntimeMoveAppliedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if strings.TrimSpace(payload.Move) == "" {
		return errors.New("move is required")
	}
	if payload.Hope == nil && payload.Stress == nil && payload.Armor == nil {
		return errors.New("downtime_move applied must change at least one state field")
	}
	return nil
}

func validateCharacterTemporaryArmorApplyPayload(raw json.RawMessage) error {
	var payload CharacterTemporaryArmorApplyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.CharacterID.String(), "character_id"); err != nil {
		return err
	}
	if err := requireTrimmedValue(payload.Source, "source"); err != nil {
		return err
	}
	if !isTemporaryArmorDuration(strings.TrimSpace(payload.Duration)) {
		return errors.New("duration must be short_rest, long_rest, session, or scene")
	}
	if payload.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	return nil
}

func validateCharacterTemporaryArmorAppliedPayload(raw json.RawMessage) error {
	return validateCharacterTemporaryArmorApplyPayload(raw)
}

func hasDamagePatchMutation(hpBefore, hpAfter, armorBefore, armorAfter *int) bool {
	return hasIntFieldChange(hpBefore, hpAfter) || hasIntFieldChange(armorBefore, armorAfter)
}

func isTemporaryArmorDuration(duration string) bool {
	switch duration {
	case "short_rest", "long_rest", "session", "scene":
		return true
	default:
		return false
	}
}
