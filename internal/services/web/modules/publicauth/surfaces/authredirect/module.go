package authredirect

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Module owns post-auth redirect routes.
type Module struct {
	inner publicauth.Module
}

// NewWithGatewayAndPolicy builds the auth redirect module with explicit gateway/policy dependencies.
func NewWithGatewayAndPolicy(gateway publicauth.AuthGateway, policy requestmeta.SchemePolicy) Module {
	return Module{inner: publicauth.NewAuthRedirectWithGatewayAndPolicy(gateway, policy)}
}

// ID returns the stable module identifier.
func (m Module) ID() string {
	return m.inner.ID()
}

// Mount returns the auth redirect route mount contract.
func (m Module) Mount() (module.Mount, error) {
	return m.inner.Mount()
}
