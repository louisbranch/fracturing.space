package notifications

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	notificationsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// CompositionConfig owns the startup wiring required to construct the
// production notifications module.
type CompositionConfig struct {
	Client notificationsgateway.NotificationClient
	Base   modulehandler.Base
}

// Compose builds the notifications module from the exact startup dependencies
// the area owns.
func Compose(config CompositionConfig) module.Module {
	gateway := notificationsgateway.NewGRPCGateway(config.Client)
	return New(Config{
		Service: notificationsapp.NewService(gateway),
		Base:    config.Base,
	})
}
