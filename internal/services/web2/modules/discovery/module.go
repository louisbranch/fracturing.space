package discovery

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
)

// Module provides public discovery routes.
type Module struct{}

// New returns a discovery module.
func New() Module { return Module{} }

// ID returns a stable module identifier.
func (Module) ID() string { return "discovery" }

// Mount wires discovery route handlers.
func (Module) Mount(module.Dependencies) (module.Mount, error) {
	mux := http.NewServeMux()
	svc := newService()
	h := newHandlers(svc)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.DiscoverPrefix, Handler: mux}, nil
}
