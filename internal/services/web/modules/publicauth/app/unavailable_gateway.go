package app

import (
	"context"
	"encoding/json"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

const authServiceUnavailableMessage = "auth service is not configured"

type unavailableGateway struct{}

// NewUnavailableGateway returns a fail-closed publicauth gateway.
func NewUnavailableGateway() Gateway {
	return unavailableGateway{}
}

func (unavailableGateway) CreateUser(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableGateway) BeginPasskeyRegistration(context.Context, string) (PasskeyChallenge, error) {
	return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableGateway) FinishPasskeyRegistration(context.Context, string, json.RawMessage) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableGateway) BeginPasskeyLogin(context.Context) (PasskeyChallenge, error) {
	return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableGateway) FinishPasskeyLogin(context.Context, string, json.RawMessage) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableGateway) CreateWebSession(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}

func (unavailableGateway) HasValidWebSession(context.Context, string) bool {
	return false
}

func (unavailableGateway) RevokeWebSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, authServiceUnavailableMessage)
}
