package systems

import (
	"net/http"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	systemsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/systems"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides systems routes.
type Module struct {
	service systemsmodule.Service
}

// New returns a systems module.
func New(service systemsmodule.Service) Module { return Module{service: service} }

// ID returns a stable module identifier.
func (Module) ID() string { return "systems" }

// Mount wires systems routes.
func (m Module) Mount() (mod.Mount, error) {
	mux := http.NewServeMux()
	if m.service == nil {
		mux.HandleFunc(routepath.SystemsPrefix, http.NotFound)
		return mod.Mount{Prefix: routepath.SystemsPrefix, Handler: mux}, nil
	}
	mux.HandleFunc(routepath.Systems, m.service.HandleSystemsPage)
	mux.HandleFunc(routepath.SystemsRows, m.service.HandleSystemsTable)
	mux.HandleFunc(routepath.SystemsPrefix, func(w http.ResponseWriter, r *http.Request) {
		systemsmodule.HandleSystemPath(w, r, m.service)
	})
	return mod.Mount{Prefix: routepath.SystemsPrefix, Handler: mux}, nil
}
