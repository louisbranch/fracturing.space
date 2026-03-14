package discovery

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides public discovery routes.
type Module struct {
	gateway discoveryapp.Gateway
}

// Config defines constructor dependencies for a discovery module.
type Config struct {
	Gateway discoveryapp.Gateway
}

// New returns a discovery module with explicit dependencies.
func New(config Config) Module {
	return Module{gateway: config.Gateway}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "discovery" }

// Healthy reports whether the discovery module has an operational gateway.
func (m Module) Healthy() bool {
	return IsGatewayHealthy(m.gateway)
}

// Mount wires discovery route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	base := publichandler.NewBase()
	h := newHandlers(base, discoveryapp.NewService(m.gateway))
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.DiscoverPrefix, Handler: mux}, nil
}
