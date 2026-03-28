package discovery

import (
	"log/slog"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
	discoverygateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/gateway"
)

// CompositionConfig owns the startup wiring required to construct the
// production discovery module.
type CompositionConfig struct {
	Client discoverygateway.DiscoveryClient
	Logger *slog.Logger
}

// Compose builds the discovery module from the exact startup dependencies the
// area owns.
func Compose(config CompositionConfig) module.Module {
	gateway := discoverygateway.NewGRPCGateway(config.Client)
	return New(Config{
		Service: discoveryapp.NewService(gateway, config.Logger),
	})
}
