package notifications

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	notificationsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// CompositionConfig owns the startup wiring required to construct the
// production notifications module without leaking gateway internals into the
// registry package.
type CompositionConfig struct {
	Base               modulehandler.Base
	NotificationClient notificationsgateway.NotificationClient
}

// Compose builds the production notifications module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	gateway := notificationsgateway.NewGRPCGateway(config.NotificationClient)
	return New(Config{
		Service: notificationsapp.NewService(gateway),
		Base:    config.Base,
		Healthy: notificationsapp.IsGatewayHealthy(gateway),
	})
}
