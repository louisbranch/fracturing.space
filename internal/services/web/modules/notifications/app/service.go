package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// service defines an internal contract used at this web package boundary.
type service struct {
	gateway Gateway
}

// NewService constructs a notifications service with fail-closed defaults.
func NewService(gateway Gateway) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway}
}

// ListNotifications returns the package view collection for this workflow.
func (s service) ListNotifications(ctx context.Context, userID string) ([]NotificationSummary, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return nil, err
	}
	items, err := s.gateway.ListNotifications(ctx, resolvedUserID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []NotificationSummary{}, nil
	}
	return items, nil
}

// GetNotification centralizes this web behavior in one helper seam.
func (s service) GetNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return NotificationSummary{}, err
	}
	resolvedNotificationID, err := requireNotificationID(notificationID)
	if err != nil {
		return NotificationSummary{}, err
	}
	item, err := s.gateway.GetNotification(ctx, resolvedUserID, resolvedNotificationID)
	if err != nil {
		return NotificationSummary{}, err
	}
	return requireNotificationFound(item)
}

// OpenNotification applies this package workflow transition.
func (s service) OpenNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return NotificationSummary{}, err
	}
	resolvedNotificationID, err := requireNotificationID(notificationID)
	if err != nil {
		return NotificationSummary{}, err
	}
	item, err := s.gateway.OpenNotification(ctx, resolvedUserID, resolvedNotificationID)
	if err != nil {
		return NotificationSummary{}, err
	}
	return requireNotificationFound(item)
}

// requireNotificationID validates inbound notification ids before gateway use.
func requireNotificationID(notificationID string) (string, error) {
	resolvedNotificationID := strings.TrimSpace(notificationID)
	if resolvedNotificationID == "" {
		return "", apperrors.E(apperrors.KindNotFound, "notification not found")
	}
	return resolvedNotificationID, nil
}

// requireNotificationFound enforces this package invariant for gateway lookups.
func requireNotificationFound(item NotificationSummary) (NotificationSummary, error) {
	if strings.TrimSpace(item.ID) == "" {
		return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
	}
	return item, nil
}
