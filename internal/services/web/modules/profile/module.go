package profile

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides public user profile routes.
type Module struct {
	gateway         ProfileGateway
	assetBaseURL    string
	resolveSignedIn module.ResolveSignedIn
}

// New returns a profile module with the given narrow dependencies.
func New(socialClient SocialClient, assetBaseURL string, resolveSignedIn module.ResolveSignedIn) Module {
	return NewWithGateway(NewGRPCGateway(socialClient), assetBaseURL, resolveSignedIn)
}

// NewWithGateway returns a profile module with an explicit gateway.
func NewWithGateway(gateway ProfileGateway, assetBaseURL string, resolveSignedIn module.ResolveSignedIn) Module {
	return Module{gateway: gateway, assetBaseURL: assetBaseURL, resolveSignedIn: resolveSignedIn}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "profile" }

// Healthy reports whether the profile module has an operational gateway.
func (m Module) Healthy() bool {
	if m.gateway == nil {
		return false
	}
	_, unavailable := m.gateway.(unavailableGateway)
	return !unavailable
}

// Mount wires public profile route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	svc := newService(m.gateway, m.assetBaseURL)
	base := publichandler.NewBase(publichandler.WithResolveViewerSignedIn(m.resolveSignedIn))
	h := newHandlers(svc, base)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.UserProfilePrefix, Handler: mux}, nil
}
