package settings

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// registerRoutes centralizes this web behavior in one helper seam.
func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	registerAccountRoutes(mux, h)
	registerAIRoutes(mux, h)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppSettingsRestPattern, h.WriteNotFound)
	mux.HandleFunc(http.MethodPost+" "+routepath.AppSettingsRestPattern, h.WriteNotFound)
}
