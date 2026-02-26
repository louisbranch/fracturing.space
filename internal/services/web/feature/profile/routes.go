package profile

import (
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Service is the profile surface contract consumed by profile route registration.
type Service interface {
	HandleProfile(w http.ResponseWriter, r *http.Request)
}

// RegisterRoutes wires profile routes into the app mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.AppProfile, service.HandleProfile)
}
