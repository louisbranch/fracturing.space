package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
)

func ValidateCountdownCreatePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CountdownCreatePayload) error {
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
	})
}

func ValidateCountdownCreatedPayload(raw json.RawMessage) error {
	return ValidateCountdownCreatePayload(raw)
}

func ValidateCountdownUpdatePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CountdownUpdatePayload) error {
		if err := RequireTrimmedValue(p.CountdownID.String(), "countdown_id"); err != nil {
			return err
		}
		if p.Before == p.After && p.Delta == 0 {
			return errors.New("countdown update must change value")
		}
		return nil
	})
}

func ValidateCountdownUpdatedPayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CountdownUpdatedPayload) error {
		return RequireTrimmedValue(p.CountdownID.String(), "countdown_id")
	})
}

func ValidateCountdownDeletePayload(raw json.RawMessage) error {
	return ValidatePayload(raw, func(p payload.CountdownDeletePayload) error {
		return RequireTrimmedValue(p.CountdownID.String(), "countdown_id")
	})
}

func ValidateCountdownDeletedPayload(raw json.RawMessage) error {
	return ValidateCountdownDeletePayload(raw)
}
