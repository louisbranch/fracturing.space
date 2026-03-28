package notifications

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	notificationsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// Compose builds the notifications module from the exact startup dependencies
// the area owns.
func Compose(client notificationsgateway.NotificationClient, base modulehandler.Base) module.Module {
	gateway := notificationsgateway.NewGRPCGateway(client)
	return New(Config{
		Service: notificationsapp.NewService(gateway),
		Base:    base,
	})
}
