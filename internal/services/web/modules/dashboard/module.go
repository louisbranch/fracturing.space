package dashboard

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated dashboard routes.
type Module struct {
	gateway        DashboardGateway
	base           modulehandler.Base
	healthProvider dashboardapp.HealthProvider
}

// New returns a dashboard module with zero-value dependencies (degraded mode).
func New() Module {
	return Module{}
}

// NewWithGateway returns a dashboard module with explicit gateway and handler dependencies.
func NewWithGateway(gateway DashboardGateway, base modulehandler.Base, healthProvider dashboardapp.HealthProvider) Module {
	return Module{gateway: gateway, base: base, healthProvider: healthProvider}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "dashboard" }

// Healthy reports whether the dashboard module has an operational gateway.
func (m Module) Healthy() bool {
	return dashboardapp.IsGatewayHealthy(m.gateway)
}

// Mount wires dashboard route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	h := newHandlers(dashboardapp.NewService(m.gateway, nil, m.healthProvider), m.base)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.DashboardPrefix, Handler: mux}, nil
}
