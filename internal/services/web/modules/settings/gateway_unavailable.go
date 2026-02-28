package settings

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type unavailableGateway struct{}

func (unavailableGateway) LoadProfile(context.Context, string) (SettingsProfile, error) {
	return SettingsProfile{}, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) SaveProfile(context.Context, string, SettingsProfile) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) LoadLocale(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) SaveLocale(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) ListAIKeys(context.Context, string) ([]SettingsAIKey, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) CreateAIKey(context.Context, string, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

func (unavailableGateway) RevokeAIKey(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}
