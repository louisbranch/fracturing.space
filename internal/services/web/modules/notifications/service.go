package notifications

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// NotificationSummary is a transport-safe summary for notification listings.
type NotificationSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Read  bool   `json:"read"`
}

// NotificationGateway loads notification summaries for web handlers.
type NotificationGateway interface {
	ListNotifications(context.Context) ([]NotificationSummary, error)
	OpenNotification(context.Context, string) (NotificationSummary, error)
}

type service struct {
	gateway NotificationGateway
}

type staticGateway struct{}

type unavailableGateway struct{}

func (staticGateway) ListNotifications(context.Context) ([]NotificationSummary, error) {
	return []NotificationSummary{{ID: "notification-1", Title: "Welcome", Read: false}}, nil
}

func (staticGateway) OpenNotification(_ context.Context, notificationID string) (NotificationSummary, error) {
	return NotificationSummary{ID: notificationID, Title: "Welcome", Read: true}, nil
}

func (unavailableGateway) ListNotifications(context.Context) ([]NotificationSummary, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "notifications service is not configured")
}

func (unavailableGateway) OpenNotification(context.Context, string) (NotificationSummary, error) {
	return NotificationSummary{}, apperrors.E(apperrors.KindUnavailable, "notifications service is not configured")
}

func newService(gateway NotificationGateway) service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway}
}

func (s service) listNotifications(ctx context.Context) ([]NotificationSummary, error) {
	items, err := s.gateway.ListNotifications(ctx)
	if err != nil {
		return nil, err
	}
	// TODO(web-ux): return an explicit empty state instead of not-found when no notifications exist.
	if len(items) == 0 {
		return nil, apperrors.E(apperrors.KindNotFound, "no notifications found")
	}
	return items, nil
}

func (s service) openNotification(ctx context.Context, notificationID string) (NotificationSummary, error) {
	item, err := s.gateway.OpenNotification(ctx, notificationID)
	if err != nil {
		return NotificationSummary{}, err
	}
	if item.ID == "" {
		return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
	}
	return item, nil
}
