package shell

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Module owns the public shell + root discovery/auth page routes.
type Module struct {
	inner publicauth.Module
}

// Config defines constructor dependencies for the shell module.
type Config struct {
	Gateway     publicauthapp.Gateway
	RequestMeta requestmeta.SchemePolicy
}

// New builds the shell module with explicit dependencies.
func New(config Config) Module {
	return Module{inner: publicauth.New(publicauth.Config{
		Gateway:     config.Gateway,
		RequestMeta: config.RequestMeta,
		Surface:     publicauth.SurfaceShell,
	})}
}

// ID returns the stable module identifier.
func (m Module) ID() string {
	return m.inner.ID()
}

// Mount returns the shell route mount contract.
func (m Module) Mount() (module.Mount, error) {
	return m.inner.Mount()
}
