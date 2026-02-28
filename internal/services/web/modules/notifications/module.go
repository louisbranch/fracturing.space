package notifications

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated notification routes.
type Module struct {
	gateway NotificationGateway
	base    modulehandler.Base
}

// New returns a notifications module with zero-value dependencies (degraded mode).
func New() Module {
	return Module{}
}

// NewWithGateway returns a notifications module with explicit gateway and handler dependencies.
func NewWithGateway(gateway NotificationGateway, base modulehandler.Base) Module {
	return Module{gateway: gateway, base: base}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "notifications" }

// Healthy reports whether the notifications module has an operational gateway.
func (m Module) Healthy() bool {
	if m.gateway == nil {
		return false
	}
	_, unavailable := m.gateway.(unavailableGateway)
	return !unavailable
}

// Mount wires notifications route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	svc := newService(m.gateway)
	h := newHandlers(svc, m.base)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.Notifications, Handler: mux}, nil
}
