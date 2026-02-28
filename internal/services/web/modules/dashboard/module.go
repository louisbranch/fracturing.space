package dashboard

import (
	"log"
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated dashboard routes.
type Module struct {
	gateway DashboardGateway
	base    modulehandler.Base
	health  []ServiceHealthEntry
}

// New returns a dashboard module with zero-value dependencies (degraded mode).
func New() Module {
	return Module{}
}

// NewWithGateway returns a dashboard module with explicit gateway and handler dependencies.
func NewWithGateway(gateway DashboardGateway, base modulehandler.Base, health []ServiceHealthEntry) Module {
	return Module{gateway: gateway, base: base, health: health}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "dashboard" }

// Healthy reports whether the dashboard module has an operational gateway.
func (m Module) Healthy() bool {
	if m.gateway == nil {
		return false
	}
	_, unavailable := m.gateway.(unavailableGateway)
	return !unavailable
}

// Mount wires dashboard route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	h := newHandlers(newService(m.gateway, log.Default(), m.health), m.base)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.DashboardPrefix, Handler: mux}, nil
}
