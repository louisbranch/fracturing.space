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
	gateway        dashboardapp.Gateway
	base           modulehandler.Base
	healthProvider dashboardapp.HealthProvider
}

// Config defines constructor dependencies for a dashboard module.
type Config struct {
	Gateway        dashboardapp.Gateway
	Base           modulehandler.Base
	HealthProvider dashboardapp.HealthProvider
}

// New returns a dashboard module with explicit dependencies.
func New(config Config) Module {
	return Module{
		gateway:        config.Gateway,
		base:           config.Base,
		healthProvider: config.HealthProvider,
	}
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
	return module.Mount{Prefix: routepath.DashboardPrefix, CanonicalRoot: true, Handler: mux}, nil
}
