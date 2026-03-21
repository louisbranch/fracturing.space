package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func ValidateDamageApplyPayload(raw json.RawMessage) error {
	var p payload.DamageApplyPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if !HasDamagePatchMutation(p.HpBefore, p.HpAfter, p.StressAfter, p.ArmorBefore, p.ArmorAfter) {
		return errors.New("damage apply must change hp, stress, or armor")
	}
	if err := ValidateDamageAdapterInvariants(p); err != nil {
		return err
	}
	return nil
}

func ValidateDamageAppliedPayload(raw json.RawMessage) error {
	var p payload.DamageAppliedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.CharacterID.String()) == "" {
		return errors.New("character_id is required")
	}
	if p.Hp == nil && p.Stress == nil && p.Armor == nil {
		return errors.New("damage applied must include hp, stress, or armor")
	}
	return ValidateDamageAppliedInvariants(p)
}

func ValidateDamageAppliedInvariants(p payload.DamageAppliedPayload) error {
	if p.ArmorSpent < 0 || p.ArmorSpent > mechanics.ArmorMaxCap {
		return fmt.Errorf("armor_spent must be in range 0..%d", mechanics.ArmorMaxCap)
	}
	if p.Marks < 0 || p.Marks > rules.MaxDamageMarks {
		return fmt.Errorf("marks must be in range 0..%d", rules.MaxDamageMarks)
	}
	if p.RollSeq != nil && *p.RollSeq == 0 {
		return errors.New("roll_seq must be positive")
	}
	if severity := strings.TrimSpace(p.Severity); severity != "" {
		switch severity {
		case "none", "minor", "major", "severe", "massive":
			// allowed
		default:
			return errors.New("severity must be one of none, minor, major, severe, massive")
		}
	}
	for _, id := range p.SourceCharacterIDs {
		if strings.TrimSpace(id.String()) == "" {
			return errors.New("source_character_ids must not contain empty values")
		}
	}
	return nil
}

func ValidateMultiTargetDamageApplyPayload(raw json.RawMessage) error {
	var p payload.MultiTargetDamageApplyPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if len(p.Targets) == 0 {
		return errors.New("targets is required and must not be empty")
	}
	for i, t := range p.Targets {
		if strings.TrimSpace(t.CharacterID.String()) == "" {
			return fmt.Errorf("targets[%d]: character_id is required", i)
		}
		if !HasDamagePatchMutation(t.HpBefore, t.HpAfter, t.StressAfter, t.ArmorBefore, t.ArmorAfter) {
			return fmt.Errorf("targets[%d]: damage apply must change hp, stress, or armor", i)
		}
		if err := ValidateDamageAdapterInvariants(t); err != nil {
			return fmt.Errorf("targets[%d]: %w", i, err)
		}
	}
	return nil
}

func ValidateAdversaryDamageApplyPayload(raw json.RawMessage) error {
	var p payload.AdversaryDamageApplyPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.AdversaryID.String()) == "" {
		return errors.New("adversary_id is required")
	}
	if !HasDamagePatchMutation(p.HpBefore, p.HpAfter, nil, p.ArmorBefore, p.ArmorAfter) {
		return errors.New("damage apply must change hp or armor")
	}
	return nil
}

func ValidateAdversaryDamageAppliedPayload(raw json.RawMessage) error {
	var p payload.AdversaryDamageAppliedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.AdversaryID.String()) == "" {
		return errors.New("adversary_id is required")
	}
	if p.Hp == nil && p.Armor == nil {
		return errors.New("damage applied must include hp or armor")
	}
	return nil
}

func ValidateDamageAdapterInvariants(p payload.DamageApplyPayload) error {
	if p.ArmorSpent < 0 || p.ArmorSpent > mechanics.ArmorMaxCap {
		return fmt.Errorf("armor_spent must be in range 0..%d", mechanics.ArmorMaxCap)
	}
	if p.Marks < 0 || p.Marks > rules.MaxDamageMarks {
		return fmt.Errorf("marks must be in range 0..%d", rules.MaxDamageMarks)
	}
	if p.RollSeq != nil && *p.RollSeq == 0 {
		return errors.New("roll_seq must be positive")
	}
	if severity := strings.TrimSpace(p.Severity); severity != "" {
		switch severity {
		case "none", "minor", "major", "severe", "massive":
			// allowed
		default:
			return errors.New("severity must be one of none, minor, major, severe, massive")
		}
	}
	for _, id := range p.SourceCharacterIDs {
		if strings.TrimSpace(id.String()) == "" {
			return errors.New("source_character_ids must not contain empty values")
		}
	}
	return nil
}

func ValidateDowntimeMoveAppliedPayload(raw json.RawMessage) error {
	var p payload.DowntimeMoveAppliedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	return ValidateDowntimeMoveAppliedPayloadFields(p)
}

func ValidateDowntimeMoveAppliedPayloadFields(p payload.DowntimeMoveAppliedPayload) error {
	if strings.TrimSpace(p.ActorCharacterID.String()) == "" {
		return errors.New("actor_character_id is required")
	}
	if strings.TrimSpace(p.Move) == "" {
		return errors.New("move is required")
	}
	if strings.TrimSpace(p.TargetCharacterID.String()) == "" &&
		p.HP == nil &&
		p.Hope == nil &&
		p.Stress == nil &&
		p.Armor == nil &&
		strings.TrimSpace(p.CountdownID.String()) == "" {
		return errors.New("downtime_move applied must target a character or countdown")
	}
	if strings.TrimSpace(p.TargetCharacterID.String()) != "" &&
		p.HP == nil &&
		p.Hope == nil &&
		p.Stress == nil &&
		p.Armor == nil &&
		strings.TrimSpace(p.CountdownID.String()) == "" {
		return errors.New("downtime_move applied target requires a state change or countdown update")
	}
	return nil
}

func ValidateCharacterTemporaryArmorApplyPayload(raw json.RawMessage) error {
	var p payload.CharacterTemporaryArmorApplyPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.CharacterID.String(), "character_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.Source, "source"); err != nil {
		return err
	}
	if !IsTemporaryArmorDuration(strings.TrimSpace(p.Duration)) {
		return errors.New("duration must be short_rest, long_rest, session, or scene")
	}
	if p.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	return nil
}

func ValidateCharacterTemporaryArmorAppliedPayload(raw json.RawMessage) error {
	return ValidateCharacterTemporaryArmorApplyPayload(raw)
}

func HasDamagePatchMutation(hpBefore, hpAfter, stressAfter, armorBefore, armorAfter *int) bool {
	return HasIntFieldChange(hpBefore, hpAfter) || stressAfter != nil || HasIntFieldChange(armorBefore, armorAfter)
}

func IsTemporaryArmorDuration(duration string) bool {
	switch duration {
	case "short_rest", "long_rest", "session", "scene":
		return true
	default:
		return false
	}
}
