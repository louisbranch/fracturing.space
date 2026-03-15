// Package provider defines shared AI provider identity values used across
// domain packages. Keeping this in one place avoids transport packages having
// to translate between duplicate provider enums.
package provider

import (
	"errors"
	"strings"
)

// Provider identifies an AI provider integration.
type Provider string

const (
	// OpenAI is the only supported AI provider in the current runtime.
	OpenAI Provider = "openai"
)

// ErrInvalid indicates a provider value is missing or unsupported.
var ErrInvalid = errors.New("provider is invalid")

// Normalize trims and validates one provider value.
func Normalize(raw string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(OpenAI):
		return OpenAI, nil
	default:
		return "", ErrInvalid
	}
}
