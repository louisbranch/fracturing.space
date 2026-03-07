package campaigns

import (
	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides campaigns routes.
type Module struct {
	handlers Handlers
}

// New returns a campaigns module.
func New(handlers Handlers) Module { return Module{handlers: handlers} }

// ID returns a stable module identifier.
func (Module) ID() string { return "campaigns" }

// Mount wires campaigns routes.
func (m Module) Mount() (mod.Mount, error) {
	return mod.Mount{
		Prefix:  routepath.CampaignsPrefix,
		Handler: newRoutes(m.handlers),
	}, nil
}
