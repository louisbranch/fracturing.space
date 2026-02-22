package icons

import (
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Service defines icon route handlers consumed by this route module.
type Service interface {
	HandleIconsPage(w http.ResponseWriter, r *http.Request)
	HandleIconsTable(w http.ResponseWriter, r *http.Request)
}

// RegisterRoutes wires icon routes into the provided mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.Icons, service.HandleIconsPage)
	mux.HandleFunc(routepath.IconsTable, service.HandleIconsTable)
}
