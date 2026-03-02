package app

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// unavailableGateway defines an internal contract used at this web package boundary.
type unavailableGateway struct{}

// NewUnavailableGateway returns a fail-closed notifications gateway.
func NewUnavailableGateway() Gateway {
	return unavailableGateway{}
}

// IsGatewayHealthy reports whether a notifications gateway is configured.
func IsGatewayHealthy(gateway Gateway) bool {
	if gateway == nil {
		return false
	}
	_, unavailable := gateway.(unavailableGateway)
	return !unavailable
}

// ListNotifications returns the package view collection for this workflow.
func (unavailableGateway) ListNotifications(context.Context, string) ([]NotificationSummary, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "notifications service is not configured")
}

// GetNotification centralizes this web behavior in one helper seam.
func (unavailableGateway) GetNotification(context.Context, string, string) (NotificationSummary, error) {
	return NotificationSummary{}, apperrors.E(apperrors.KindUnavailable, "notifications service is not configured")
}

// OpenNotification applies this package workflow transition.
func (unavailableGateway) OpenNotification(context.Context, string, string) (NotificationSummary, error) {
	return NotificationSummary{}, apperrors.E(apperrors.KindUnavailable, "notifications service is not configured")
}
