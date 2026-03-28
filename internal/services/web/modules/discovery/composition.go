package discovery

import (
	"log/slog"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
	discoverygateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/gateway"
)

// Compose builds the discovery module from the exact startup dependencies the
// area owns.
func Compose(client discoverygateway.DiscoveryClient, logger *slog.Logger) module.Module {
	gateway := discoverygateway.NewGRPCGateway(client)
	return New(Config{
		Service: discoveryapp.NewService(gateway, logger),
	})
}
