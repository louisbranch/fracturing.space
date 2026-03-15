package discovery

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides public discovery routes.
type Module struct {
	service discoveryapp.Service
	healthy bool
}

// Config defines constructor dependencies for a discovery module.
type Config struct {
	Service discoveryapp.Service
	Healthy bool
}

// New returns a discovery module with explicit dependencies.
func New(config Config) Module {
	service := config.Service
	if service == nil {
		service = discoveryapp.NewService(nil)
	}
	return Module{service: service, healthy: config.Healthy}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "discovery" }

// Healthy reports whether the discovery module has an operational runtime
// service backing its transport surface.
func (m Module) Healthy() bool {
	return m.healthy
}

// Mount wires discovery route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	h := newHandlers(m.service)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.DiscoverPrefix, CanonicalRoot: true, Handler: mux}, nil
}
