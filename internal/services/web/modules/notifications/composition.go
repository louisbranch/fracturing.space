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

// ProtectedSurfaceOptions carries the cross-cutting inputs the protected registry is
// allowed to pass into notifications composition.
type ProtectedSurfaceOptions struct {
	Base modulehandler.Base
}

// Compose builds the production notifications module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	gateway := notificationsgateway.NewGRPCGateway(config.NotificationClient)
	return New(Config{
		Service: notificationsapp.NewService(gateway),
		Base:    config.Base,
	})
}

// ComposeProtected composes the notifications protected surface when the owning
// dependency set is complete. The registry can use this to gate optional surface
// selection through one owned constructor.
func ComposeProtected(options ProtectedSurfaceOptions, deps Dependencies) (module.Module, bool) {
	if !deps.configured() {
		return nil, false
	}
	return Compose(newCompositionConfig(options, deps)), true
}

// newCompositionConfig projects startup dependencies into notifications
// composition input.
func newCompositionConfig(options ProtectedSurfaceOptions, deps Dependencies) CompositionConfig {
	return CompositionConfig{
		Base:               options.Base,
		NotificationClient: deps.NotificationClient,
	}
}

// configured reports whether the notifications dependency set has the client required
// for production-safe mounting.
func (deps Dependencies) configured() bool {
	return deps.NotificationClient != nil
}
