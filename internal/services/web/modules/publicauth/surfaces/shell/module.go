package shell

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Module owns the public shell + root discovery/auth page routes.
type Module struct {
	inner publicauth.Module
}

// NewWithGatewayAndPolicy builds the shell module with explicit gateway/policy dependencies.
func NewWithGatewayAndPolicy(gateway publicauth.AuthGateway, policy requestmeta.SchemePolicy) Module {
	return Module{inner: publicauth.NewShellWithGatewayAndPolicy(gateway, policy)}
}

// ID returns the stable module identifier.
func (m Module) ID() string {
	return m.inner.ID()
}

// Mount returns the shell route mount contract.
func (m Module) Mount() (module.Mount, error) {
	return m.inner.Mount()
}
