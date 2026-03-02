package icons

import (
	"net/http"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	iconsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/icons"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides icons routes.
type Module struct {
	service iconsmodule.Service
}

// New returns an icons module.
func New(service iconsmodule.Service) Module { return Module{service: service} }

// ID returns a stable module identifier.
func (Module) ID() string { return "icons" }

// Mount wires icons routes.
func (m Module) Mount() (mod.Mount, error) {
	mux := http.NewServeMux()
	if m.service == nil {
		mux.HandleFunc(routepath.IconsPrefix, http.NotFound)
		return mod.Mount{Prefix: routepath.IconsPrefix, Handler: mux}, nil
	}
	mux.HandleFunc(routepath.Icons, m.service.HandleIconsPage)
	mux.HandleFunc(routepath.IconsRows, m.service.HandleIconsTable)
	return mod.Mount{Prefix: routepath.IconsPrefix, Handler: mux}, nil
}
