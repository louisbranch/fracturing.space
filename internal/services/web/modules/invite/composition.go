package invite

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	inviteapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/app"
	invitegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// CompositionConfig owns the startup wiring required to construct the
// production invite module without leaking gateway internals into the registry
// package.
type CompositionConfig struct {
	RequestMeta   requestmeta.SchemePolicy
	Principal     principal.PrincipalResolver
	DashboardSync dashboardsync.Service

	InviteClient invitegateway.InviteClient
	AuthClient   invitegateway.AuthClient
}

// PublicSurfaceOptions carries the shared cross-cutting inputs the public
// registry is allowed to pass into invite composition.
type PublicSurfaceOptions struct {
	RequestMeta   requestmeta.SchemePolicy
	Principal     principal.PrincipalResolver
	DashboardSync dashboardsync.Service
}

// Compose builds the production invite module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	gateway := invitegateway.NewGRPCGateway(config.InviteClient, config.AuthClient)
	return New(Config{
		Service:       inviteapp.NewService(gateway),
		RequestMeta:   config.RequestMeta,
		Principal:     config.Principal,
		DashboardSync: config.DashboardSync,
		Healthy:       gateway != nil,
	})
}

// ComposePublic composes the public invite surface from the invite-owned
// dependency bundle plus request-scoped cross-cutting options.
func ComposePublic(options PublicSurfaceOptions, deps Dependencies) module.Module {
	return Compose(CompositionConfig{
		RequestMeta:   options.RequestMeta,
		Principal:     options.Principal,
		DashboardSync: options.DashboardSync,
		InviteClient:  deps.InviteClient,
		AuthClient:    deps.AuthClient,
	})
}
