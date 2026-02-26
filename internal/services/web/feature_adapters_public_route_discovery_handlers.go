package web

import (
	"net/http"

	discoveryfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/discovery"
)

func (h *handler) publicDiscoveryRouteHandlers() discoveryfeature.Handlers {
	return discoveryfeature.Handlers{
		Discover: func(w http.ResponseWriter, r *http.Request) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			discoveryfeature.HandleDiscover(h.publicDiscoveryRouteDependencies(w, r), w, r)
		},
		DiscoverCampaign: func(w http.ResponseWriter, r *http.Request) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			discoveryfeature.HandleDiscoverCampaign(h.publicDiscoveryRouteDependencies(w, r), w, r)
		},
	}
}
