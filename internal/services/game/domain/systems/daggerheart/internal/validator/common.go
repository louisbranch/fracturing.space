package validator

import (
	"errors"
	"fmt"
	"strings"
)

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
