package notifications

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
)

// Module provides authenticated notification routes.
type Module struct {
	gateway NotificationGateway
}

// New returns a notifications module.
func New() Module {
	return Module{}
}

// NewWithGateway returns a notifications module with an explicit gateway dependency.
func NewWithGateway(gateway NotificationGateway) Module {
	return Module{gateway: gateway}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "notifications" }

// Mount wires notifications route handlers.
func (m Module) Mount(deps module.Dependencies) (module.Mount, error) {
	mux := http.NewServeMux()
	gateway := m.gateway
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	svc := newService(gateway)
	h := newHandlers(svc, deps)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.Notifications, Handler: mux}, nil
}
