package discovery

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// Module provides public discovery routes.
type Module struct{}

// New returns a discovery module.
func New() Module { return Module{} }

// ID returns a stable module identifier.
func (Module) ID() string { return "discovery" }

// Mount wires discovery route handlers.
func (Module) Mount() (module.Mount, error) {
	mux := http.NewServeMux()
	base := publichandler.NewBase()
	h := newHandlers(base)
	registerRoutes(mux, h)
	return module.Mount{Prefix: routepath.DiscoverPrefix, Handler: mux}, nil
}
