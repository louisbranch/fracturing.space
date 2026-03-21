package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ValidatePayload unmarshals raw JSON into a typed payload and runs check.
// This eliminates the repeated unmarshal+error boilerplate from every
// Validate*Payload function.
func ValidatePayload[P any](raw json.RawMessage, check func(P) error) error {
	var p P
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	return check(p)
}

// RequireCharacterID validates that a character_id field is present and non-empty.
func RequireCharacterID(id fmt.Stringer) error {
	return RequireTrimmedValue(id.String(), "character_id")
}

// RequireAdversaryID validates that an adversary_id field is present and non-empty.
func RequireAdversaryID(id fmt.Stringer) error {
	return RequireTrimmedValue(id.String(), "adversary_id")
}

// RequireTrimmedValue validates a required string field after whitespace trim.
func RequireTrimmedValue(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(field + " is required")
	}
	return nil
}

// RequirePositive validates strictly positive integer fields.
func RequirePositive(value int, field string) error {
	if value <= 0 {
		return errors.New(field + " must be positive")
	}
	return nil
}

// RequireRange validates inclusive numeric bounds.
func RequireRange(value, min, max int, field string) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be in range %d..%d", field, min, max)
	}
	return nil
}
