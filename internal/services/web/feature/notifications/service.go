package notifications

import (
	"net/http"

	routing "github.com/louisbranch/fracturing.space/internal/services/web/feature/routing"
)

// Handlers configures callback-backed notifications service construction.
type Handlers struct {
	Notifications    http.HandlerFunc
	NotificationOpen routing.StringParamHandler
}

type callbackService struct {
	handlers Handlers
}

// NewService builds a notifications Service backed by handler callbacks.
func NewService(handlers Handlers) Service {
	return callbackService{handlers: handlers}
}

func (s callbackService) HandleNotifications(w http.ResponseWriter, r *http.Request) {
	routing.CallOrNotFound(w, r, s.handlers.Notifications)
}

func (s callbackService) HandleNotificationOpen(w http.ResponseWriter, r *http.Request, notificationID string) {
	routing.CallStringOrNotFound(w, r, s.handlers.NotificationOpen, notificationID)
}
