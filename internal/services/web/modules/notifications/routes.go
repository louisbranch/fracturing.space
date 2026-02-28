package notifications

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppNotifications, h.handleIndex)
	mux.HandleFunc(http.MethodGet+" "+routepath.Notifications, h.handleIndex)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppNotificationPattern, h.handleDetailRoute)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppNotificationOpenPattern, httpx.MethodNotAllowed(http.MethodPost))
	mux.HandleFunc(http.MethodPost+" "+routepath.AppNotificationOpenPattern, h.handleOpenRoute)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppNotificationRestPattern, h.WriteNotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppNotificationRestPattern, h.WriteNotFound)
}
