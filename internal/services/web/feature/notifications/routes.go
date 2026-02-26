package notifications

import (
	"net/http"

	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
	routing "github.com/louisbranch/fracturing.space/internal/services/web/feature/routing"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Service is the notifications surface contract consumed by notifications route registration.
type Service interface {
	HandleNotifications(w http.ResponseWriter, r *http.Request)
	HandleNotificationOpen(w http.ResponseWriter, r *http.Request, notificationID string)
}

// RegisterRoutes wires notifications routes into the app mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.AppNotifications, service.HandleNotifications)
	mux.HandleFunc(routepath.AppNotificationsPrefix, func(w http.ResponseWriter, r *http.Request) {
		HandleNotificationSubpath(w, r, service)
	})
}

// HandleNotificationSubpath parses notification routes and dispatches to handlers.
func HandleNotificationSubpath(w http.ResponseWriter, r *http.Request, service Service) {
	if service == nil {
		http.NotFound(w, r)
		return
	}
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}
	notificationID, ok := routing.SingleSegment(r.URL.Path, routepath.AppNotificationsPrefix)
	if !ok {
		http.NotFound(w, r)
		return
	}
	service.HandleNotificationOpen(w, r, notificationID)
}
