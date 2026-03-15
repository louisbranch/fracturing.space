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
	service notificationsapp.Service
	base    modulehandler.Base
	healthy bool
}

// Config defines constructor dependencies for a notifications module.
type Config struct {
	Service notificationsapp.Service
	Base    modulehandler.Base
	Healthy bool
}

// New returns a notifications module with explicit dependencies.
func New(config Config) Module {
	service := config.Service
	if service == nil {
		service = notificationsapp.NewService(nil)
	}
	return Module{service: service, base: config.Base, healthy: config.Healthy}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "notifications" }

// Healthy reports whether the notifications module has an operational runtime
// service backing its transport surface.
func (m Module) Healthy() bool {
	return m.healthy
}

// Mount wires notifications route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	h := newHandlers(m.service, m.base)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.Notifications, CanonicalRoot: true, Handler: mux}, nil
}
