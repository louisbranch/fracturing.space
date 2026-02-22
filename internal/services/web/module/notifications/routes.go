package notifications

import (
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Service is the notifications surface contract consumed by notifications route registration.
type Service interface {
	HandleNotifications(w http.ResponseWriter, r *http.Request)
	HandleNotificationsSubroutes(w http.ResponseWriter, r *http.Request)
}

// RegisterRoutes wires notifications routes into the app mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.AppNotifications, service.HandleNotifications)
	mux.HandleFunc(routepath.AppNotificationsPrefix, service.HandleNotificationsSubroutes)
}
