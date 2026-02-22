package httpmux

import (
	"io/fs"
	"net/http"

	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// MountStatic wires static asset serving into the root mux.
func MountStatic(rootMux *http.ServeMux, staticFS fs.FS, withStaticMime func(http.Handler) http.Handler) {
	if rootMux == nil || staticFS == nil {
		return
	}
	staticHandler := http.StripPrefix(routepath.StaticPrefix, http.FileServer(http.FS(staticFS)))
	if withStaticMime != nil {
		staticHandler = withStaticMime(staticHandler)
	}
	rootMux.Handle(routepath.StaticPrefix, staticHandler)
}

// MountAdminRoutes mounts admin application routes under root path.
func MountAdminRoutes(rootMux *http.ServeMux, adminMux *http.ServeMux) {
	if rootMux == nil || adminMux == nil {
		return
	}
	rootMux.Handle(routepath.Root, adminMux)
}
