// Package provider defines the shared AI provider vocabulary.
//
// It owns provider identity values plus the shared OAuth, invoke, model, and
// usage contracts that transport, domain, and concrete adapters consume.
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
