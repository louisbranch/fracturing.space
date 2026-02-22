package dashboard

import (
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Service defines dashboard route handlers consumed by this route module.
type Service interface {
	HandleDashboard(w http.ResponseWriter, r *http.Request)
	HandleDashboardContent(w http.ResponseWriter, r *http.Request)
}

// RegisterRoutes wires dashboard routes into the provided mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.Root, service.HandleDashboard)
	mux.HandleFunc(routepath.DashboardContent, service.HandleDashboardContent)
}
