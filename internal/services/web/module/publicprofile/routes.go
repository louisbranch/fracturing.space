package publicprofile

import (
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Service is the public profile surface contract consumed by route registration.
type Service interface {
	HandlePublicProfile(w http.ResponseWriter, r *http.Request)
}

// RegisterRoutes wires public profile routes into the public mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.UserProfilePrefix, service.HandlePublicProfile)
}
