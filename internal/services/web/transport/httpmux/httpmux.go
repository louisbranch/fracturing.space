package httpmux

import (
	"io/fs"
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// MountStatic wires the shared static route into the root mux.
func MountStatic(rootMux *http.ServeMux, staticFS fs.FS, withStaticMime func(http.Handler) http.Handler) {
	if rootMux == nil || staticFS == nil {
		return
	}
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))
	if withStaticMime != nil {
		staticHandler = withStaticMime(staticHandler)
	}
	rootMux.Handle("/static/", staticHandler)
}

// MountAppAndPublicRoutes wires app and public route groups under root.
func MountAppAndPublicRoutes(rootMux *http.ServeMux, appMux *http.ServeMux, publicMux *http.ServeMux) {
	if rootMux == nil {
		return
	}
	if appMux != nil {
		rootMux.Handle(routepath.AppRoot, appMux)
		rootMux.Handle(routepath.AppRootPrefix, appMux)
	}
	if publicMux != nil {
		rootMux.Handle("/", publicMux)
	}
}
