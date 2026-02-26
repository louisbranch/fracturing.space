package profile

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
)

// Module provides authenticated profile routes.
type Module struct {
	gateway ProfileGateway
}

// New returns a profile module.
func New() Module {
	return Module{}
}

// NewWithGateway returns a profile module with an explicit gateway dependency.
func NewWithGateway(gateway ProfileGateway) Module {
	return Module{gateway: gateway}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "profile" }

// Mount wires profile route handlers.
func (m Module) Mount(deps module.Dependencies) (module.Mount, error) {
	mux := http.NewServeMux()
	gateway := m.gateway
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	svc := newService(gateway)
	h := newHandlers(svc, deps)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.ProfilePrefix, Handler: mux}, nil
}
