package discovery

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
)

func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.DiscoverPrefix+"{$}", h.handleIndex)
	mux.HandleFunc(http.MethodGet+" "+routepath.DiscoverPrefix+"{rest...}", h.handleIndex)
}
