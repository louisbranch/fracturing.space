package discovery

import (
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Service is the public discovery surface contract consumed by route registration.
type Service interface {
	HandleDiscover(w http.ResponseWriter, r *http.Request)
	HandleDiscoverCampaign(w http.ResponseWriter, r *http.Request)
}

// RegisterRoutes wires discovery routes into the public mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.Discover, service.HandleDiscover)
	mux.HandleFunc(routepath.DiscoverCampaignsPrefix, service.HandleDiscoverCampaign)
}
