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
	MessageType string     `json:"message_type"`
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

// requireUserID validates and returns a trimmed user ID, or returns an
// unauthorized error if it is blank.
func requireUserID(userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", apperrors.EK(apperrors.KindUnauthorized, "error.web.message.user_id_is_required", "user id is required")
	}
	return userID, nil
}

type service struct {
	gateway NotificationGateway
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
