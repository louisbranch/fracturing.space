package profile

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides public user profile routes.
type Module struct {
	gateway      profileapp.Gateway
	assetBaseURL string
	principal    requestresolver.PrincipalResolver
}

// Config defines constructor dependencies for a profile module.
type Config struct {
	Gateway      profileapp.Gateway
	AssetBaseURL string
	Principal    requestresolver.PrincipalResolver
}

// New returns a profile module with explicit dependencies.
func New(config Config) Module {
	return Module{
		gateway:      config.Gateway,
		assetBaseURL: config.AssetBaseURL,
		principal:    config.Principal,
	}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "profile" }

// Healthy reports whether the profile module has an operational gateway.
func (m Module) Healthy() bool {
	return profileapp.IsGatewayHealthy(m.gateway)
}

// Mount wires public profile route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	svc := profileapp.NewService(m.gateway)
	base := publichandler.NewBaseFromPrincipal(m.principal)
	h := newHandlers(svc, m.assetBaseURL, base)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.UserProfilePrefix, Handler: mux}, nil
}
