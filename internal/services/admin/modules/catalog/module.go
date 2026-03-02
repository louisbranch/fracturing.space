package catalog

import (
	"net/http"
	"strings"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	catalogmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides catalog routes.
type Module struct {
	service catalogmodule.Service
}

// New returns a catalog module.
func New(service catalogmodule.Service) Module { return Module{service: service} }

// ID returns a stable module identifier.
func (Module) ID() string { return "catalog" }

// Mount wires catalog routes.
func (m Module) Mount() (mod.Mount, error) {
	mux := http.NewServeMux()
	if m.service == nil {
		mux.HandleFunc(routepath.CatalogPrefix, http.NotFound)
		return mod.Mount{Prefix: routepath.CatalogPrefix, Handler: mux}, nil
	}
	mux.HandleFunc(routepath.Catalog, m.service.HandleCatalogPage)
	mux.HandleFunc(routepath.CatalogPrefix, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/table") {
			http.NotFound(w, r)
			return
		}
		catalogmodule.HandleCatalogPath(w, r, m.service)
	})
	return mod.Mount{Prefix: routepath.CatalogPrefix, Handler: mux}, nil
}
