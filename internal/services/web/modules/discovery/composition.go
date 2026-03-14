package discovery

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	discoverygateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/gateway"
)

// CompositionConfig owns the startup wiring required to construct the
// production discovery module without leaking gateway internals into the
// registry package.
type CompositionConfig struct {
	DiscoveryClient discoverygateway.DiscoveryClient
}

// Compose builds the production discovery module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	return New(Config{
		Gateway: discoverygateway.NewGRPCGateway(config.DiscoveryClient),
	})
}
