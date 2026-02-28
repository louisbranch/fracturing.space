package passkeys

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Module owns passkey JSON endpoint routes.
type Module struct {
	inner publicauth.Module
}

// NewWithGatewayAndPolicy builds the passkeys module with explicit gateway/policy dependencies.
func NewWithGatewayAndPolicy(gateway publicauth.AuthGateway, policy requestmeta.SchemePolicy) Module {
	return Module{inner: publicauth.NewPasskeysWithGatewayAndPolicy(gateway, policy)}
}

// ID returns the stable module identifier.
func (m Module) ID() string {
	return m.inner.ID()
}

// Mount returns the passkeys route mount contract.
func (m Module) Mount() (module.Mount, error) {
	return m.inner.Mount()
}
