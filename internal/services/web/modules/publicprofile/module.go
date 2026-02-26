package publicprofile

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides public user profile routes.
type Module struct{}

// New returns a public profile module.
func New() Module { return Module{} }

// ID returns a stable module identifier.
func (Module) ID() string { return "publicprofile" }

// Mount wires public profile route handlers.
func (Module) Mount(module.Dependencies) (module.Mount, error) {
	mux := http.NewServeMux()
	svc := newService()
	h := newHandlers(svc)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.UserProfilePrefix, Handler: mux}, nil
}
