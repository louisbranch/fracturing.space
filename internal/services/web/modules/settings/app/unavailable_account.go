package app

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// LoadProfile loads the package state needed for this request path.
func (unavailableGateway) LoadProfile(context.Context, string) (SettingsProfile, error) {
	return SettingsProfile{}, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// SaveProfile centralizes this web behavior in one helper seam.
func (unavailableGateway) SaveProfile(context.Context, string, SettingsProfile) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// LoadLocale loads the package state needed for this request path.
func (unavailableGateway) LoadLocale(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// SaveLocale centralizes this web behavior in one helper seam.
func (unavailableGateway) SaveLocale(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}
