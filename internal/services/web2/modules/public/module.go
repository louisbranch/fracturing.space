package public

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
)

// Module provides unauthenticated root/auth routes.
type Module struct{}

// New returns a public/auth module.
func New() Module {
	return Module{}
}

// ID returns a stable identifier for diagnostics and startup logs.
func (Module) ID() string {
	return "public"
}

// Mount wires public routes under the auth/root prefix.
func (Module) Mount(deps module.Dependencies) (module.Mount, error) {
	mux := http.NewServeMux()
	svc := newService(deps)
	h := newHandlers(svc)
	registerRoutes(mux, h)
	return module.Mount{Prefix: "/", Handler: mux}, nil
}
