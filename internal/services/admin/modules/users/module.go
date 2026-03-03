package users

import (
	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

// Module provides users routes.
type Module struct {
	service Service
}

// New returns a users module.
func New(service Service) Module { return Module{service: service} }

// ID returns a stable module identifier.
func (Module) ID() string { return "users" }

// Mount wires users routes.
func (m Module) Mount() (mod.Mount, error) {
	return mod.Mount{
		Prefix:  routepath.UsersPrefix,
		Handler: newRoutes(m.service),
	}, nil
}
