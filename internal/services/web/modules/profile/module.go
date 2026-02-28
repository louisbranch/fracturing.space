package profile

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides public user profile routes.
type Module struct{}

// New returns a profile module.
func New() Module { return Module{} }

// ID returns a stable module identifier.
func (Module) ID() string { return "profile" }

// Mount wires public profile route handlers.
func (Module) Mount(deps module.Dependencies) (module.Mount, error) {
	mux := http.NewServeMux()
	svc := newService(newGRPCGateway(deps), deps.AssetBaseURL)
	h := newHandlers(svc, deps)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.UserProfilePrefix, Handler: mux}, nil
}
