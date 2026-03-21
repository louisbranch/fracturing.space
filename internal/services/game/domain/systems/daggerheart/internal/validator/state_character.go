package validator

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func ValidateCharacterProfileReplacePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p daggerheartstate.CharacterProfileReplacePayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		return p.Profile.Validate()
	})
}

func ValidateCharacterProfileReplacedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p daggerheartstate.CharacterProfileReplacedPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		return p.Profile.Validate()
	})
}

func ValidateCharacterProfileDeletePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p daggerheartstate.CharacterProfileDeletePayload) error {
		return RequireCharacterID(p.CharacterID)
	})
}

func ValidateCharacterProfileDeletedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p daggerheartstate.CharacterProfileDeletedPayload) error {
		return RequireCharacterID(p.CharacterID)
	})
}

func ValidateCharacterStatePatchPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CharacterStatePatchPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		if !HasCharacterStateChange(p) {
			return errors.New("character_state patch must change at least one field")
		}
		return nil
	})
}

func ValidateCharacterStatePatchedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CharacterStatePatchedPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		if p.HP == nil && p.Hope == nil && p.HopeMax == nil &&
			p.Stress == nil && p.Armor == nil && p.LifeState == nil &&
			p.ClassState == nil && p.SubclassState == nil && p.ImpenetrableUsedThisShortRest == nil {
			return errors.New("character_state_patched must include at least one after field")
		}
		return nil
	})
}

func ValidateClassFeatureApplyPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.ClassFeatureApplyPayload) error {
		if err := RequireTrimmedValue(p.ActorCharacterID.String(), "actor_character_id"); err != nil {
			return err
		}
		if err := RequireTrimmedValue(p.Feature, "feature"); err != nil {
			return err
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
	})
}

func ValidateSubclassFeatureApplyPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.SubclassFeatureApplyPayload) error {
		if err := RequireTrimmedValue(p.ActorCharacterID.String(), "actor_character_id"); err != nil {
			return err
		}
		if err := RequireTrimmedValue(p.Feature, "feature"); err != nil {
			return err
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
	})
}
