package discovery

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides public discovery routes.
type Module struct {
	gateway Gateway
}

// New returns a discovery module with no discovery backend (fail-closed).
func New() Module { return Module{} }

// NewWithGateway returns a discovery module backed by the given gateway.
func NewWithGateway(gw Gateway) Module {
	return Module{gateway: gw}
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
	h := newHandlers(base, m.gateway)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.DiscoverPrefix, Handler: mux}, nil
}
