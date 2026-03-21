package app

import "context"

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
