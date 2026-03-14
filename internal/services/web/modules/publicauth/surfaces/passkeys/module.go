package passkeys

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Module owns passkey JSON endpoint routes.
type Module struct {
	inner publicauth.Module
}

// Config defines constructor dependencies for the passkeys module.
type Config struct {
	Gateway     publicauthapp.Gateway
	RequestMeta requestmeta.SchemePolicy
	AuthBaseURL string
}

// New builds the passkeys module with explicit dependencies.
func New(config Config) Module {
	return Module{inner: publicauth.New(publicauth.Config{
		Gateway:     config.Gateway,
		RequestMeta: config.RequestMeta,
		AuthBaseURL: config.AuthBaseURL,
		Surface:     publicauth.SurfacePasskeys,
	})}
}

// ID returns the stable module identifier.
func (m Module) ID() string {
	return m.inner.ID()
}

// Mount returns the passkeys route mount contract.
func (m Module) Mount() (module.Mount, error) {
	return m.inner.Mount()
}
