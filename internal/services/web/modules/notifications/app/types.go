package app

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
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

// Gateway loads notification summaries for web handlers.
type Gateway interface {
	ListNotifications(context.Context, string) ([]NotificationSummary, error)
	GetNotification(context.Context, string, string) (NotificationSummary, error)
	OpenNotification(context.Context, string, string) (NotificationSummary, error)
}

// Service exposes notification orchestration methods used by transport handlers.
type Service interface {
	ListNotifications(context.Context, string) ([]NotificationSummary, error)
	GetNotification(context.Context, string, string) (NotificationSummary, error)
	OpenNotification(context.Context, string, string) (NotificationSummary, error)
}

// RequireUserID validates and returns a trimmed user ID, or returns an
// unauthorized error if it is blank.
func RequireUserID(userID string) (string, error) {
	return userid.Require(userID)
}
