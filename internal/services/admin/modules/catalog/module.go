package catalog

import (
	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

const (
	// DaggerheartSystemID is the currently supported system slug in admin catalog routes.
	DaggerheartSystemID = "daggerheart"
)

// Module provides catalog routes.
type Module struct {
	handlers Handlers
}

// New returns a catalog module.
func New(handlers Handlers) Module { return Module{handlers: handlers} }

// ID returns a stable module identifier.
func (Module) ID() string { return "catalog" }

// Mount wires catalog routes.
func (m Module) Mount() (mod.Mount, error) {
	return mod.Mount{
		Prefix:  routepath.CatalogPrefix,
		Handler: newRoutes(m.handlers),
	}, nil
}
