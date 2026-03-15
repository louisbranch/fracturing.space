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
func NewUnavailableGateway() unavailableGateway {
	return unavailableGateway{}
}

// BeginAccountRegistration fails fast because signup cannot run without auth.
func (unavailableGateway) BeginAccountRegistration(context.Context, string) (PasskeyChallenge, error) {
	return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// CheckUsernameAvailability fails fast because signup validation requires auth.
func (unavailableGateway) CheckUsernameAvailability(context.Context, string) (UsernameAvailability, error) {
	return UsernameAvailability{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// FinishAccountRegistration fails fast because signup cannot complete without auth.
func (unavailableGateway) FinishAccountRegistration(context.Context, string, json.RawMessage) (PasskeyRegistrationReveal, error) {
	return PasskeyRegistrationReveal{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// AcknowledgeAccountRegistration fails fast because staged signup cannot activate without auth.
func (unavailableGateway) AcknowledgeAccountRegistration(context.Context, string, string) (PasskeyFinish, error) {
	return PasskeyFinish{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// BeginPasskeyLogin fails fast because login cannot start without auth.
func (unavailableGateway) BeginPasskeyLogin(context.Context, string) (PasskeyChallenge, error) {
	return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// FinishPasskeyLogin fails fast because login cannot complete without auth.
func (unavailableGateway) FinishPasskeyLogin(context.Context, string, json.RawMessage, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// BeginAccountRecovery fails fast because recovery cannot start without auth.
func (unavailableGateway) BeginAccountRecovery(context.Context, string, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// BeginRecoveryPasskeyRegistration fails fast because recovery cannot continue without auth.
func (unavailableGateway) BeginRecoveryPasskeyRegistration(context.Context, string) (PasskeyChallenge, error) {
	return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// FinishRecoveryPasskeyRegistration fails fast because recovery cannot complete without auth.
func (unavailableGateway) FinishRecoveryPasskeyRegistration(context.Context, string, string, json.RawMessage, string) (PasskeyFinish, error) {
	return PasskeyFinish{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// CreateWebSession fails fast because there is no auth service to mint sessions.
func (unavailableGateway) CreateWebSession(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// RevokeWebSession fails fast so callers can surface missing auth wiring.
func (unavailableGateway) RevokeWebSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}
