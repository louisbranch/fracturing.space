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
	service dashboardapp.Service
	base    modulehandler.Base
}

// Config defines constructor dependencies for a dashboard module.
type Config struct {
	Service dashboardapp.Service
	Base    modulehandler.Base
}

// New returns a dashboard module with explicit dependencies.
func New(config Config) Module {
	service := config.Service
	if service == nil {
		service = dashboardapp.NewService(nil, nil, nil)
	}
	return Module{
		service: service,
		base:    config.Base,
	}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "dashboard" }

// Mount wires dashboard route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	h := newHandlers(m.service, m.base)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.DashboardPrefix, CanonicalRoot: true, Handler: mux}, nil
}
