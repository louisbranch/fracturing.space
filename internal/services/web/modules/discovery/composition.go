package discovery

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
	discoverygateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/gateway"
)

// CompositionConfig owns the startup wiring required to construct the
// production discovery module without leaking gateway internals into the
// registry package.
type CompositionConfig struct {
	DiscoveryClient discoverygateway.DiscoveryClient
}

// PublicSurfaceOptions carries shared cross-cutting inputs the public registry is
// allowed to pass into discovery composition.
type PublicSurfaceOptions struct{}

// Compose builds the production discovery module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	gateway := discoverygateway.NewGRPCGateway(config.DiscoveryClient)
	return New(Config{
		Service: discoveryapp.NewService(gateway),
	})
}

// ComposePublic composes the discovery public surface when required dependencies
// are available. The registry can use this to keep optional public routes out
// of the mounted module set instead of keeping a fail-closed fallback.
func ComposePublic(_ PublicSurfaceOptions, deps Dependencies) (module.Module, bool) {
	if !deps.configured() {
		return nil, false
	}
	return Compose(newCompositionConfig(deps)), true
}

// newCompositionConfig projects startup dependencies into discovery composition
// input.
func newCompositionConfig(deps Dependencies) CompositionConfig {
	return CompositionConfig{
		DiscoveryClient: deps.DiscoveryClient,
	}
}

// configured reports whether the discovery dependency set has the client required
// for production-safe mounting.
func (deps Dependencies) configured() bool {
	return deps.DiscoveryClient != nil
}
