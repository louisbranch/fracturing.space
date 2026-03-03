package icons

import (
	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides icons routes.
type Module struct {
	service Service
}

// New returns an icons module.
func New(service Service) Module { return Module{service: service} }

// ID returns a stable module identifier.
func (Module) ID() string { return "icons" }

// Mount wires icons routes.
func (m Module) Mount() (mod.Mount, error) {
	return mod.Mount{
		Prefix:  routepath.IconsPrefix,
		Handler: newRoutes(m.service),
	}, nil
}
