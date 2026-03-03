package systems

import (
	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides systems routes.
type Module struct {
	service Service
}

// New returns a systems module.
func New(service Service) Module { return Module{service: service} }

// ID returns a stable module identifier.
func (Module) ID() string { return "systems" }

// Mount wires systems routes.
func (m Module) Mount() (mod.Mount, error) {
	return mod.Mount{
		Prefix:  routepath.SystemsPrefix,
		Handler: newRoutes(m.service),
	}, nil
}
