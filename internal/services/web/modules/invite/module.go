package invite

import (
	"context"
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// DashboardSync exposes dashboard refresh hooks needed after invite actions.
type DashboardSync interface {
	InviteChanged(context.Context, []string, string)
}

// Module provides public invite landing routes.
type Module struct {
	gateway       inviteapp.Gateway
	base          publichandler.Base
	requestMeta   requestmeta.SchemePolicy
	resolveUserID module.ResolveUserID
	sync          DashboardSync
}

// Config defines constructor dependencies for the invite module.
type Config struct {
	Gateway       inviteapp.Gateway
	Base          publichandler.Base
	RequestMeta   requestmeta.SchemePolicy
	ResolveUserID module.ResolveUserID
	DashboardSync DashboardSync
}

// New returns an invite module with explicit dependencies.
func New(config Config) Module {
	return Module{
		gateway:       config.Gateway,
		base:          config.Base,
		requestMeta:   config.RequestMeta,
		resolveUserID: config.ResolveUserID,
		sync:          config.DashboardSync,
	}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "invite" }

// Healthy reports whether the invite module has an operational gateway.
func (m Module) Healthy() bool {
	return m.gateway != nil
}

// Mount wires public invite route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	h := newHandlers(inviteapp.NewService(m.gateway), m.base, m.requestMeta, m.resolveUserID, m.sync)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.InvitePrefix, Handler: mux}, nil
}
