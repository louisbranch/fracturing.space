package profile

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides public user profile routes.
type Module struct {
	service      profileapp.Service
	assetBaseURL string
	principal    principal.PrincipalResolver
}

// Config defines constructor dependencies for a profile module.
type Config struct {
	Service      profileapp.Service
	AssetBaseURL string
	Principal    principal.PrincipalResolver
}

// New returns a profile module with explicit dependencies.
func New(config Config) Module {
	service := config.Service
	if service == nil {
		service = profileapp.NewService(nil)
	}
	return Module{
		service:      service,
		assetBaseURL: config.AssetBaseURL,
		principal:    config.Principal,
	}
}

// ID returns a stable module identifier.
func (Module) ID() string { return "profile" }

// Mount wires public profile route handlers.
func (m Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	base := publichandler.NewBaseFromPrincipal(m.principal)
	h := newHandlers(m.service, m.assetBaseURL, base)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.UserProfilePrefix, Handler: mux}, nil
}
