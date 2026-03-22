package validator

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func validateCountdownCreatePayload(p payload.SceneCountdownCreatePayload) error {
	if err := RequireTrimmedValue(p.CountdownID.String(), "countdown_id"); err != nil {
		return err
	}
	if err := RequireTrimmedValue(p.Name, "name"); err != nil {
		return err
	}
	if _, err := rules.NormalizeCountdownTone(p.Tone); err != nil {
		return err
	}
	if _, err := rules.NormalizeCountdownAdvancementPolicy(p.AdvancementPolicy); err != nil {
		return err
	}
	if _, err := rules.NormalizeCountdownLoopBehavior(p.LoopBehavior); err != nil {
		return err
	}
	if _, err := rules.NormalizeCountdownStatus(p.Status); err != nil {
		return err
	}
	if err := RequirePositive(p.StartingValue, "starting_value"); err != nil {
		return err
	}
	if p.RemainingValue < 0 || p.RemainingValue > p.StartingValue {
		return fmt.Errorf("remaining_value must be in range 0..%d", p.StartingValue)
	}
	if p.StartingRoll != nil {
		if p.StartingRoll.Min <= 0 || p.StartingRoll.Max < p.StartingRoll.Min {
			return errors.New("starting_roll range is invalid")
		}
		if p.StartingRoll.Value < p.StartingRoll.Min || p.StartingRoll.Value > p.StartingRoll.Max {
			return errors.New("starting_roll value is out of range")
		}
	}
	return nil
}

func ValidateSceneCountdownCreatePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, validateCountdownCreatePayload)
}

func ValidateSceneCountdownCreatedPayload(raw json.RawMessage) error {
	return ValidateSceneCountdownCreatePayload(raw)
}

func ValidateCampaignCountdownCreatePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CampaignCountdownCreatePayload) error {
		return validateCountdownCreatePayload(payload.SceneCountdownCreatePayload(p))
	})
}

func ValidateCampaignCountdownCreatedPayload(raw json.RawMessage) error {
	return ValidateCampaignCountdownCreatePayload(raw)
}

func validateCountdownAdvancePayload(p payload.SceneCountdownAdvancePayload) error {
	if err := RequireTrimmedValue(p.CountdownID.String(), "countdown_id"); err != nil {
		return err
	}
	if p.BeforeRemaining < 0 || p.AfterRemaining < 0 {
		return errors.New("countdown remaining values must be non-negative")
	}
	if p.AdvancedBy <= 0 {
		return errors.New("advanced_by must be positive")
	}
	if p.BeforeRemaining == p.AfterRemaining && p.StatusBefore == p.StatusAfter && !p.Triggered {
		return errors.New("countdown advance must record a state change")
	}
	return nil
}

func ValidateSceneCountdownAdvancePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, validateCountdownAdvancePayload)
}

func ValidateSceneCountdownAdvancedPayload(raw json.RawMessage) error {
	return ValidateSceneCountdownAdvancePayload(raw)
}

func ValidateCampaignCountdownAdvancePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CampaignCountdownAdvancePayload) error {
		return validateCountdownAdvancePayload(payload.SceneCountdownAdvancePayload(p))
	})
}

func ValidateCampaignCountdownAdvancedPayload(raw json.RawMessage) error {
	return ValidateCampaignCountdownAdvancePayload(raw)
}

func validateCountdownTriggerResolvePayload(p payload.SceneCountdownTriggerResolvePayload) error {
	if err := RequireTrimmedValue(p.CountdownID.String(), "countdown_id"); err != nil {
		return err
	}
	return nil
}

func ValidateSceneCountdownTriggerResolvePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, validateCountdownTriggerResolvePayload)
}

func ValidateSceneCountdownTriggerResolvedPayload(raw json.RawMessage) error {
	return ValidateSceneCountdownTriggerResolvePayload(raw)
}

func ValidateCampaignCountdownTriggerResolvePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CampaignCountdownTriggerResolvePayload) error {
		return validateCountdownTriggerResolvePayload(payload.SceneCountdownTriggerResolvePayload(p))
	})
}

func ValidateCampaignCountdownTriggerResolvedPayload(raw json.RawMessage) error {
	return ValidateCampaignCountdownTriggerResolvePayload(raw)
}

func ValidateSceneCountdownDeletePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.SceneCountdownDeletePayload) error {
		return RequireTrimmedValue(p.CountdownID.String(), "countdown_id")
	})
}

func ValidateSceneCountdownDeletedPayload(raw json.RawMessage) error {
	return ValidateSceneCountdownDeletePayload(raw)
}

func ValidateCampaignCountdownDeletePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CampaignCountdownDeletePayload) error {
		return RequireTrimmedValue(p.CountdownID.String(), "countdown_id")
	})
}

func ValidateCampaignCountdownDeletedPayload(raw json.RawMessage) error {
	return ValidateCampaignCountdownDeletePayload(raw)
}
