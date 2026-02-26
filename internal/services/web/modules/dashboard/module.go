package dashboard

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated dashboard routes.
type Module struct{}

// New returns a dashboard module.
func New() Module {
	return Module{}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "dashboard" }

// Mount wires dashboard route handlers.
func (Module) Mount(deps module.Dependencies) (module.Mount, error) {
	mux := http.NewServeMux()
	h := newHandlers(newService(), deps)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.DashboardPrefix, Handler: mux}, nil
}
