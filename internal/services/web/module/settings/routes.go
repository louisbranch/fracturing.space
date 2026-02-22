package settings

import (
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Service is the settings surface contract consumed by settings route registration.
type Service interface {
	HandleSettings(w http.ResponseWriter, r *http.Request)
	HandleSettingsSubroutes(w http.ResponseWriter, r *http.Request)
}

// RegisterRoutes wires settings routes into the app mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.AppSettings, service.HandleSettings)
	mux.HandleFunc(routepath.AppSettingsPrefix, service.HandleSettingsSubroutes)
}
