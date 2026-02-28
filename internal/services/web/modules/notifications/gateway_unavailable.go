package notifications

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type unavailableGateway struct{}

func (unavailableGateway) ListNotifications(context.Context, string) ([]NotificationSummary, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "notifications service is not configured")
}

func (unavailableGateway) GetNotification(context.Context, string, string) (NotificationSummary, error) {
	return NotificationSummary{}, apperrors.E(apperrors.KindUnavailable, "notifications service is not configured")
}

func (unavailableGateway) OpenNotification(context.Context, string, string) (NotificationSummary, error) {
	return NotificationSummary{}, apperrors.E(apperrors.KindUnavailable, "notifications service is not configured")
}
