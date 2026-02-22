package notifications

import (
	"net/http"

	moduleruntime "github.com/louisbranch/fracturing.space/internal/services/web/module/runtime"
)

// Handlers configures callback-backed notifications service construction.
type Handlers struct {
	Notifications    http.HandlerFunc
	NotificationOpen moduleruntime.StringParamHandler
}

type callbackService struct {
	handlers Handlers
}

// NewService builds a notifications Service backed by handler callbacks.
func NewService(handlers Handlers) Service {
	return callbackService{handlers: handlers}
}

func (s callbackService) HandleNotifications(w http.ResponseWriter, r *http.Request) {
	moduleruntime.CallOrNotFound(w, r, s.handlers.Notifications)
}

func (s callbackService) HandleNotificationOpen(w http.ResponseWriter, r *http.Request, notificationID string) {
	moduleruntime.CallStringOrNotFound(w, r, s.handlers.NotificationOpen, notificationID)
}
