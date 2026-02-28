package settings

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Option configures a settings module.
type Option func(*Module)

// WithGateway sets the settings gateway.
func WithGateway(g SettingsGateway) Option {
	return func(m *Module) { m.gateway = g }
}

// WithBase sets the handler base for authenticated routes.
func WithBase(b modulehandler.Base) Option {
	return func(m *Module) { m.base = b }
}

// WithSchemePolicy sets the request scheme policy for cookie handling.
func WithSchemePolicy(p requestmeta.SchemePolicy) Option {
	return func(m *Module) { m.flashMeta = p }
}

// Module provides authenticated settings routes.
type Module struct {
	gateway   SettingsGateway
	base      modulehandler.Base
	flashMeta requestmeta.SchemePolicy
}

// New returns a settings module configured by the given options.
// Without options the module starts in degraded mode.
func New(opts ...Option) Module {
	var m Module
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// ID returns a stable module identifier.
func (Module) ID() string { return "settings" }

// Healthy reports whether the settings module has an operational gateway.
func (m Module) Healthy() bool {
	if m.gateway == nil {
		return false
	}
	_, unavailable := m.gateway.(unavailableGateway)
	return !unavailable
}

// Mount wires settings route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	svc := newService(m.gateway)
	h := newHandlers(svc, m.base, m.flashMeta)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.SettingsPrefix, Handler: mux}, nil
}
