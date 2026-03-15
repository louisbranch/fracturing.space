package invite

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// DashboardSync keeps public invite mutations aligned with dashboard freshness.
type DashboardSync = dashboardsync.Service

// Module provides public invite landing routes.
type Module struct {
	service     inviteapp.Service
	requestMeta requestmeta.SchemePolicy
	principal   principal.PrincipalResolver
	sync        DashboardSync
	healthy     bool
}

// Config defines constructor dependencies for the invite module.
type Config struct {
	Service       inviteapp.Service
	RequestMeta   requestmeta.SchemePolicy
	Principal     principal.PrincipalResolver
	DashboardSync DashboardSync
	Healthy       bool
}

// New returns an invite module with explicit dependencies.
func New(config Config) Module {
	service := config.Service
	if service == nil {
		service = inviteapp.NewService(nil)
	}
	sync := config.DashboardSync
	if sync == nil {
		sync = dashboardsync.Noop{}
	}
	return Module{
		service:     service,
		requestMeta: config.RequestMeta,
		principal:   config.Principal,
		sync:        sync,
		healthy:     config.Healthy,
	}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "invite" }

// Healthy reports whether the invite module has an operational runtime service
// backing its transport surface.
func (m Module) Healthy() bool {
	return m.healthy
}

// Mount wires public invite route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	h := newHandlers(m.service, m.principal, m.requestMeta, m.sync)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.InvitePrefix, Handler: mux}, nil
}
