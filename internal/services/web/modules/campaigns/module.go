package campaigns

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides authenticated campaign workspace routes.
type Module struct {
	gateway CampaignGateway
	surface routeSurface
}

// New returns a campaigns module.
func New() Module {
	return Module{surface: routeSurfaceFull}
}

// NewWithGateway returns a campaigns module with an explicit gateway dependency.
func NewWithGateway(gateway CampaignGateway) Module {
	return Module{gateway: gateway, surface: routeSurfaceFull}
}

// NewStableWithGateway returns a campaigns module with stable route exposure.
func NewStableWithGateway(gateway CampaignGateway) Module {
	return Module{gateway: gateway, surface: routeSurfaceStable}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "campaigns" }

// Mount wires campaign route handlers.
func (m Module) Mount(deps module.Dependencies) (module.Mount, error) {
	mux := http.NewServeMux()
	gateway := m.gateway
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	svc := newService(gateway)
	h := newHandlers(svc, deps)
	if m.surface == routeSurfaceStable {
		registerStableRoutes(mux, h)
	} else {
		registerRoutes(mux, h)
	}
	return module.Mount{Prefix: routepath.CampaignsPrefix, Handler: mux}, nil
}
