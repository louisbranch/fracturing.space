package campaigns

import (
	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides campaigns routes.
type Module struct {
	service Service
}

// New returns a campaigns module.
func New(service Service) Module { return Module{service: service} }

// ID returns a stable module identifier.
func (Module) ID() string { return "campaigns" }

// Mount wires campaigns routes.
func (m Module) Mount() (mod.Mount, error) {
	return mod.Mount{
		Prefix:  routepath.CampaignsPrefix,
		Handler: newRoutes(m.service),
	}, nil
}
