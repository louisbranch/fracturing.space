package validator

import (
	"encoding/json"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func ValidateBeastformTransformPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.BeastformTransformPayload) error {
		if err := RequireTrimmedValue(p.ActorCharacterID.String(), "actor_character_id"); err != nil {
			return err
		}
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		if err := RequireTrimmedValue(p.BeastformID, "beastform_id"); err != nil {
			return err
		}
		if !HasIntFieldChange(p.HopeBefore, p.HopeAfter) &&
			!HasIntFieldChange(p.StressBefore, p.StressAfter) &&
			!HasClassStateFieldChange(p.ClassStateBefore, p.ClassStateAfter) {
			return errors.New("beastform transform must change at least one field")
		}
		return nil
	})
}

func ValidateBeastformDropPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.BeastformDropPayload) error {
		if err := RequireTrimmedValue(p.ActorCharacterID.String(), "actor_character_id"); err != nil {
			return err
		}
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		if err := RequireTrimmedValue(p.BeastformID, "beastform_id"); err != nil {
			return err
		}
		if !HasClassStateFieldChange(p.ClassStateBefore, p.ClassStateAfter) {
			return errors.New("beastform drop must change class state")
		}
		return nil
	})
}

func ValidateBeastformTransformedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.BeastformTransformedPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		if err := RequireTrimmedValue(p.BeastformID, "beastform_id"); err != nil {
			return err
		}
		if daggerheartstate.NormalizedActiveBeastformPtr(p.ActiveBeastform) == nil {
			return errors.New("active_beastform is required")
		}
		return nil
	})
}

func ValidateBeastformDroppedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.BeastformDroppedPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		return RequireTrimmedValue(p.BeastformID, "beastform_id")
	})
}

func ValidateCompanionExperienceBeginPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CompanionExperienceBeginPayload) error {
		if err := RequireTrimmedValue(p.ActorCharacterID.String(), "actor_character_id"); err != nil {
			return err
		}
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		if err := RequireTrimmedValue(p.ExperienceID, "experience_id"); err != nil {
			return err
		}
		if !HasCompanionStateFieldChange(p.CompanionStateBefore, p.CompanionStateAfter) {
			return errors.New("companion begin must change companion state")
		}
		return nil
	})
}

func ValidateCompanionReturnPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CompanionReturnPayload) error {
		if err := RequireTrimmedValue(p.ActorCharacterID.String(), "actor_character_id"); err != nil {
			return err
		}
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		if err := RequireTrimmedValue(p.Resolution, "resolution"); err != nil {
			return err
		}
		if !HasIntFieldChange(p.StressBefore, p.StressAfter) &&
			!HasCompanionStateFieldChange(p.CompanionStateBefore, p.CompanionStateAfter) {
			return errors.New("companion return must change at least one field")
		}
		return nil
	})
}

func ValidateCompanionExperienceBegunPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CompanionExperienceBegunPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		if err := RequireTrimmedValue(p.ExperienceID, "experience_id"); err != nil {
			return err
		}
		if daggerheartstate.NormalizedCompanionStatePtr(p.CompanionState) == nil {
			return errors.New("companion_state is required")
		}
		return nil
	})
}

func ValidateCompanionReturnedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CompanionReturnedPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		if err := RequireTrimmedValue(p.Resolution, "resolution"); err != nil {
			return err
		}
		if daggerheartstate.NormalizedCompanionStatePtr(p.CompanionState) == nil {
			return errors.New("companion_state is required")
		}
		return nil
	})
}

func ValidateHopeSpendPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.HopeSpendPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
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
	})
}

func ValidateStressSpendPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.StressSpendPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
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
	})
}
