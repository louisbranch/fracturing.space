package daggerheart

import (
	"errors"
	"fmt"
	"strings"
)

// requireTrimmedValue validates a required string field after whitespace trim.
func requireTrimmedValue(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(field + " is required")
	}
	return nil
}

// requirePositive validates strictly positive integer fields.
func requirePositive(value int, field string) error {
	if value <= 0 {
		return errors.New(field + " must be positive")
	}
	return nil
}

// requireRange validates inclusive numeric bounds.
func requireRange(value, min, max int, field string) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be in range %d..%d", field, min, max)
	}
	return nil
}
