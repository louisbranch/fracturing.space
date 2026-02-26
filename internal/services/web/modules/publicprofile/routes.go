package publicprofile

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.UserProfilePrefix+"{$}", h.handleIndex)
	mux.HandleFunc(http.MethodGet+" "+routepath.UserProfilePrefix+"{rest...}", h.handleIndex)
}
