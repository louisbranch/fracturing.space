package notifications

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
)

func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppNotifications, h.handleIndex)
	mux.HandleFunc(http.MethodGet+" "+routepath.Notifications, h.handleIndex)
	mux.HandleFunc(http.MethodGet+" "+routepath.Notifications+"{notificationID}", h.handleOpenRoute)
	mux.HandleFunc(http.MethodGet+" "+routepath.Notifications+"{notificationID}/{rest...}", h.handleNotFound)
}
