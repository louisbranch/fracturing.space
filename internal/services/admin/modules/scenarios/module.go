package scenarios

import (
	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides scenarios routes.
type Module struct {
	handlers Handlers
}

// New returns a scenarios module.
func New(handlers Handlers) Module { return Module{handlers: handlers} }

// ID returns a stable module identifier.
func (Module) ID() string { return "scenarios" }

// Mount wires scenarios routes.
func (m Module) Mount() (mod.Mount, error) {
	return mod.Mount{
		Prefix:  routepath.ScenariosPrefix,
		Handler: newRoutes(m.handlers),
	}, nil
}
