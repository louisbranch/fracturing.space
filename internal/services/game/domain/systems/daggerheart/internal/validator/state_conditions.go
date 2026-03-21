package validator

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func ValidateConditionChangePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.ConditionChangePayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		return ValidateConditionSetPayload(
			p.ConditionsBefore,
			p.ConditionsAfter,
			p.Added,
			p.Removed,
		)
	})
}

func ValidateConditionChangedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.ConditionChangedPayload) error {
		if err := RequireCharacterID(p.CharacterID); err != nil {
			return err
		}
		if p.Conditions == nil {
			return errors.New("conditions_after is required")
		}
		if _, err := rules.NormalizeConditionStates(p.Conditions); err != nil {
			return fmt.Errorf("conditions_after: %w", err)
		}
		return nil
	})
}

func ValidateAdversaryConditionChangePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.AdversaryConditionChangePayload) error {
		if err := RequireAdversaryID(p.AdversaryID); err != nil {
			return err
		}
		return ValidateConditionSetPayload(
			p.ConditionsBefore,
			p.ConditionsAfter,
			p.Added,
			p.Removed,
		)
	})
}

func ValidateAdversaryConditionChangedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.AdversaryConditionChangedPayload) error {
		if err := RequireAdversaryID(p.AdversaryID); err != nil {
			return err
		}
		if p.Conditions == nil {
			return errors.New("conditions_after is required")
		}
		if _, err := rules.NormalizeConditionStates(p.Conditions); err != nil {
			return fmt.Errorf("conditions_after: %w", err)
		}
		return nil
	})
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
