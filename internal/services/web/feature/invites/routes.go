package invites

import (
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Service is the user-invites surface contract consumed by invites route registration.
type Service interface {
	HandleInvites(w http.ResponseWriter, r *http.Request)
	HandleInviteClaim(w http.ResponseWriter, r *http.Request)
}

// RegisterRoutes wires invite list and claim routes into the app mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.AppInvites, service.HandleInvites)
	mux.HandleFunc(routepath.AppInviteClaim, service.HandleInviteClaim)
}
