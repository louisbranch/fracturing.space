package app

import (
	"context"
	"encoding/json"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// ListPasskeys returns unavailable while settings is degraded.
func (unavailableGateway) ListPasskeys(context.Context, string) ([]SettingsPasskey, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// BeginPasskeyRegistration returns unavailable while settings is degraded.
func (unavailableGateway) BeginPasskeyRegistration(context.Context, string) (PasskeyChallenge, error) {
	return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// FinishPasskeyRegistration returns unavailable while settings is degraded.
func (unavailableGateway) FinishPasskeyRegistration(context.Context, string, json.RawMessage) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}
