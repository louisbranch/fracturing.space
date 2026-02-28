package notifications

import (
	"context"
	"strings"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// NotificationSummary is a transport-safe summary for notification listings.
type NotificationSummary struct {
	ID          string     `json:"id"`
	Topic       string     `json:"topic"`
	PayloadJSON string     `json:"payload_json"`
	Source      string     `json:"source"`
	Read        bool       `json:"read"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
}

// NotificationGateway loads notification summaries for web handlers.
type NotificationGateway interface {
	ListNotifications(context.Context, string) ([]NotificationSummary, error)
	GetNotification(context.Context, string, string) (NotificationSummary, error)
	OpenNotification(context.Context, string, string) (NotificationSummary, error)
}

type service struct {
	gateway NotificationGateway
}

type staticGateway struct{}

type unavailableGateway struct{}

func (staticGateway) ListNotifications(context.Context, string) ([]NotificationSummary, error) {
	now := time.Now().UTC()
	return []NotificationSummary{{
		ID:          "notification-1",
		Topic:       "auth.onboarding.welcome",
		PayloadJSON: `{"signup_method":"passkey"}`,
		Source:      "system",
		Read:        false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}}, nil
}

func (g staticGateway) GetNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	items, err := g.ListNotifications(ctx, userID)
	if err != nil {
		return NotificationSummary{}, err
	}
	for _, item := range items {
		if strings.TrimSpace(item.ID) == strings.TrimSpace(notificationID) {
			return item, nil
		}
	}
	return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
}

func (g staticGateway) OpenNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	item, err := g.GetNotification(ctx, userID, notificationID)
	if err != nil {
		return NotificationSummary{}, err
	}
	now := time.Now().UTC()
	item.Read = true
	item.ReadAt = &now
	item.UpdatedAt = now
	return item, nil
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

func newService(gateway NotificationGateway) service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway}
}

func (s service) listNotifications(ctx context.Context, userID string) ([]NotificationSummary, error) {
	resolvedUserID, err := requireUserID(userID)
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

func (s service) getNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return NotificationSummary{}, err
	}
	resolvedNotificationID := strings.TrimSpace(notificationID)
	if resolvedNotificationID == "" {
		return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
	}
	item, err := s.gateway.GetNotification(ctx, resolvedUserID, resolvedNotificationID)
	if err != nil {
		return NotificationSummary{}, err
	}
	if strings.TrimSpace(item.ID) == "" {
		return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
	}
	return item, nil
}

func (s service) openNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return NotificationSummary{}, err
	}
	resolvedNotificationID := strings.TrimSpace(notificationID)
	if resolvedNotificationID == "" {
		return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
	}
	item, err := s.gateway.OpenNotification(ctx, resolvedUserID, resolvedNotificationID)
	if err != nil {
		return NotificationSummary{}, err
	}
	if strings.TrimSpace(item.ID) == "" {
		return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
	}
	return item, nil
}

func requireUserID(userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", apperrors.EK(apperrors.KindUnauthorized, "error.web.message.user_id_is_required", "user id is required")
	}
	return userID, nil
}
