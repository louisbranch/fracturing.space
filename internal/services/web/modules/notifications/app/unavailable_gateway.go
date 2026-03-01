package app

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

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

func (unavailableGateway) ListNotifications(context.Context, string) ([]NotificationSummary, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "notifications service is not configured")
}

func (unavailableGateway) GetNotification(context.Context, string, string) (NotificationSummary, error) {
	return NotificationSummary{}, apperrors.E(apperrors.KindUnavailable, "notifications service is not configured")
}

func (unavailableGateway) OpenNotification(context.Context, string, string) (NotificationSummary, error) {
	return NotificationSummary{}, apperrors.E(apperrors.KindUnavailable, "notifications service is not configured")
}
