package profile

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func registerRoutes(mux *http.ServeMux, h handlers) {
	if mux == nil {
		return
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.UserProfilePattern, h.handleProfile)
	mux.HandleFunc(http.MethodGet+" "+routepath.UserProfilePrefix+"{$}", h.handleNotFound)
	mux.HandleFunc(http.MethodGet+" "+routepath.UserProfileRestPattern, h.handleNotFound)
}
