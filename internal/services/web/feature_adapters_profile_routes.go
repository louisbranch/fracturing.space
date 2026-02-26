package web

import (
	"net/http"

	featureprofile "github.com/louisbranch/fracturing.space/internal/services/web/feature/profile"
)

func (h *handler) appProfileRouteHandlers() featureprofile.Handlers {
	return featureprofile.Handlers{
		Profile: func(w http.ResponseWriter, r *http.Request) {
			if h == nil {
				http.NotFound(w, r)
				return
			}
			featureprofile.HandleAppProfile(h.appProfileRouteDependencies(w, r), w, r)
		},
	}
}
