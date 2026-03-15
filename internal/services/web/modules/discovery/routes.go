package discovery

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// registerRoutes centralizes this web behavior in one helper seam.
func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.Discover, h.handleIndex)
	mux.HandleFunc(http.MethodGet+" "+routepath.DiscoverPrefix+"{rest...}", h.handleIndex)
}
