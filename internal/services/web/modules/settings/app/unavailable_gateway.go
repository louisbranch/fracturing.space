package app

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// unavailableGateway defines an internal contract used at this web package boundary.
type unavailableGateway struct{}

// NewUnavailableGateway returns a fail-closed settings gateway.
func NewUnavailableGateway() Gateway {
	return unavailableGateway{}
}

// IsGatewayHealthy reports whether a settings gateway is configured and usable.
func IsGatewayHealthy(gateway Gateway) bool {
	if gateway == nil {
		return false
	}
	_, unavailable := gateway.(unavailableGateway)
	return !unavailable
}

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

// ListAIKeys returns the package view collection for this workflow.
func (unavailableGateway) ListAIKeys(context.Context, string) ([]SettingsAIKey, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// CreateAIKey executes package-scoped creation behavior for this flow.
func (unavailableGateway) CreateAIKey(context.Context, string, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// RevokeAIKey applies this package workflow transition.
func (unavailableGateway) RevokeAIKey(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}
