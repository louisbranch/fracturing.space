package notifications

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated notification routes.
type Module struct {
	gateway notificationsapp.Gateway
	base    modulehandler.Base
}

// Config defines constructor dependencies for a notifications module.
type Config struct {
	Gateway notificationsapp.Gateway
	Base    modulehandler.Base
}

// New returns a notifications module with explicit dependencies.
func New(config Config) Module {
	return Module{gateway: config.Gateway, base: config.Base}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "notifications" }

// Healthy reports whether the notifications module has an operational gateway.
func (m Module) Healthy() bool {
	return notificationsapp.IsGatewayHealthy(m.gateway)
}

// Mount wires notifications route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	svc := notificationsapp.NewService(m.gateway)
	h := newHandlers(svc, m.base)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.Notifications, Handler: mux}, nil
}
