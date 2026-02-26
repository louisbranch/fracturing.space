package web

import (
	"net/http"

	featurenotifications "github.com/louisbranch/fracturing.space/internal/services/web/feature/notifications"
)

func (h *handler) appNotificationsRouteHandlers() featurenotifications.Handlers {
	return featurenotifications.Handlers{
		Notifications: func(w http.ResponseWriter, r *http.Request) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			featurenotifications.HandleAppNotifications(h.appNotificationsRouteDependencies(w, r), w, r)
		},
		NotificationOpen: func(w http.ResponseWriter, r *http.Request, notificationID string) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			featurenotifications.HandleAppNotificationOpen(h.appNotificationOpenRouteDependencies(w, r), w, r, notificationID)
		},
	}
}
