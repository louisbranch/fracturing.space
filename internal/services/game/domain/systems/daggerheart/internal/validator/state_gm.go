package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func ValidateGMFearSetPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.GMFearSetPayload) error {
		if p.After == nil {
			return errors.New("after is required")
		}
		return RequireRange(*p.After, daggerheartstate.GMFearMin, daggerheartstate.GMFearMax, "after")
	})
}

func ValidateGMFearChangedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.GMFearChangedPayload) error {
		if p.Value < daggerheartstate.GMFearMin || p.Value > daggerheartstate.GMFearMax {
			return fmt.Errorf("value must be in range %d..%d", daggerheartstate.GMFearMin, daggerheartstate.GMFearMax)
		}
		return nil
	})
}

func ValidateGMMoveApplyPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.GMMoveApplyPayload) error {
		if p.FearSpent <= 0 {
			return errors.New("fear_spent must be greater than zero")
		}
		return ValidateGMMoveTarget(p.Target)
	})
}

func ValidateGMMoveAppliedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.GMMoveAppliedPayload) error {
		if p.FearSpent <= 0 {
			return errors.New("fear_spent must be greater than zero")
		}
		return ValidateGMMoveTarget(p.Target)
	})
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
