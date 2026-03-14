package app

import (
	"context"
	"encoding/json"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

const authServiceUnavailableMessage = "Auth service is not configured."

// unavailableGateway preserves consistent unavailable errors when auth is not wired.
type unavailableGateway struct{}

// NewUnavailableGateway provides a gateway fallback for startup and tests.
func NewUnavailableGateway() Gateway {
	return unavailableGateway{}
}

// BeginAccountRegistration fails fast because signup cannot run without auth.
func (unavailableGateway) BeginAccountRegistration(context.Context, string) (PasskeyChallenge, error) {
	return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// FinishAccountRegistration fails fast because signup cannot complete without auth.
func (unavailableGateway) FinishAccountRegistration(context.Context, string, json.RawMessage) (PasskeyFinish, error) {
	return PasskeyFinish{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// BeginPasskeyLogin fails fast because login cannot start without auth.
func (unavailableGateway) BeginPasskeyLogin(context.Context, string) (PasskeyChallenge, error) {
	return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// FinishPasskeyLogin fails fast because login cannot complete without auth.
func (unavailableGateway) FinishPasskeyLogin(context.Context, string, json.RawMessage) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// CreateWebSession fails fast because there is no auth service to mint sessions.
func (unavailableGateway) CreateWebSession(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// HasValidWebSession reports false because unavailable auth cannot validate sessions.
func (unavailableGateway) HasValidWebSession(context.Context, string) bool {
	return false
}

// RevokeWebSession fails fast so callers can surface missing auth wiring.
func (unavailableGateway) RevokeWebSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}
