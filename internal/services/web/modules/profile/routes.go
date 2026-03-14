package profile

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// registerRoutes centralizes this web behavior in one helper seam.
func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.UserProfilePattern, h.withUsername(h.handleProfile))
	mux.HandleFunc(http.MethodGet+" "+routepath.UserProfilePrefix+"{$}", h.handleNotFound)
	mux.HandleFunc(http.MethodGet+" "+routepath.UserProfileRestPattern, h.handleNotFound)
}
