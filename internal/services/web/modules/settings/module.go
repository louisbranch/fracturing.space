package settings

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated settings routes.
type Module struct {
	gateway SettingsGateway
}

// New returns a settings module.
func New() Module {
	return Module{}
}

// NewWithGateway returns a settings module with an explicit gateway dependency.
func NewWithGateway(gateway SettingsGateway) Module {
	return Module{gateway: gateway}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "settings" }

// Mount wires settings route handlers.
func (m Module) Mount(deps module.Dependencies) (module.Mount, error) {
	mux := http.NewServeMux()
	gateway := m.gateway
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	svc := newService(gateway)
	h := newHandlers(svc, deps)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.SettingsPrefix, Handler: mux}, nil
}
