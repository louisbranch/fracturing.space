package dashboard

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppDashboard, h.handleIndex)
	mux.HandleFunc(http.MethodGet+" "+routepath.DashboardPrefix+"{$}", h.handleIndex)
	mux.HandleFunc(http.MethodGet+" "+routepath.DashboardPrefix+"{rest...}", h.WriteNotFound)
}
