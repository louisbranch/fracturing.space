package dashboard

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated dashboard routes.
type Module struct {
	gateway DashboardGateway
}

// New returns a dashboard module.
func New() Module {
	return Module{}
}

// NewWithGateway returns a dashboard module with an explicit gateway dependency.
func NewWithGateway(gateway DashboardGateway) Module {
	return Module{gateway: gateway}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "dashboard" }

// Mount wires dashboard route handlers.
func (m Module) Mount(deps module.Dependencies) (module.Mount, error) {
	mux := http.NewServeMux()
	gateway := m.gateway
	if gateway == nil {
		gateway = NewGRPCGateway(deps)
	}
	h := newHandlers(newService(gateway), deps)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.DashboardPrefix, Handler: mux}, nil
}
