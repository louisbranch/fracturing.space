package app

import (
	"context"
	"encoding/json"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

const authServiceUnavailableMessage = "auth service is not configured"

// unavailableGateway defines an internal contract used at this web package boundary.
type unavailableGateway struct{}

// NewUnavailableGateway returns a fail-closed publicauth gateway.
func NewUnavailableGateway() Gateway {
	return unavailableGateway{}
}

// CreateUser executes package-scoped creation behavior for this flow.
func (unavailableGateway) CreateUser(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// BeginPasskeyRegistration centralizes this web behavior in one helper seam.
func (unavailableGateway) BeginPasskeyRegistration(context.Context, string) (PasskeyChallenge, error) {
	return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// FinishPasskeyRegistration centralizes this web behavior in one helper seam.
func (unavailableGateway) FinishPasskeyRegistration(context.Context, string, json.RawMessage) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// BeginPasskeyLogin centralizes this web behavior in one helper seam.
func (unavailableGateway) BeginPasskeyLogin(context.Context) (PasskeyChallenge, error) {
	return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// FinishPasskeyLogin centralizes this web behavior in one helper seam.
func (unavailableGateway) FinishPasskeyLogin(context.Context, string, json.RawMessage) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// CreateWebSession executes package-scoped creation behavior for this flow.
func (unavailableGateway) CreateWebSession(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

// HasValidWebSession reports whether this package condition is satisfied.
func (unavailableGateway) HasValidWebSession(context.Context, string) bool {
	return false
}

// RevokeWebSession applies this package workflow transition.
func (unavailableGateway) RevokeWebSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}
