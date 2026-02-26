package web

import (
	"net/http"

	publicprofilefeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/publicprofile"
)

func (h *handler) publicProfileRouteHandlers() publicprofilefeature.Handlers {
	return publicprofilefeature.Handlers{
		PublicProfile: func(w http.ResponseWriter, r *http.Request) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			publicprofilefeature.HandlePublicProfile(h.publicProfileRouteDependencies(w, r), w, r)
		},
	}
}
