package authredirect

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Module owns post-auth redirect routes.
type Module struct {
	inner publicauth.Module
}

// Config defines constructor dependencies for the auth redirect module.
type Config struct {
	Gateway     publicauthapp.Gateway
	RequestMeta requestmeta.SchemePolicy
}

// New builds the auth redirect module with explicit dependencies.
func New(config Config) Module {
	return Module{inner: publicauth.New(publicauth.Config{
		Gateway:     config.Gateway,
		RequestMeta: config.RequestMeta,
		Surface:     publicauth.SurfaceAuthRedirect,
	})}
}

// ID returns the stable module identifier.
func (m Module) ID() string {
	return m.inner.ID()
}

// Mount returns the auth redirect route mount contract.
func (m Module) Mount() (module.Mount, error) {
	return m.inner.Mount()
}
