package invite

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	invitegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
)

// CompositionConfig owns the startup wiring required to construct the
// production invite module without leaking gateway internals into the registry
// package.
type CompositionConfig struct {
	RequestMeta requestmeta.SchemePolicy
	Principal   requestresolver.PrincipalResolver

	InviteClient invitegateway.InviteClient
	AuthClient   invitegateway.AuthClient

	UserHubControlClient dashboardsync.UserHubControlClient
	GameEventClient      dashboardsync.GameEventClient
}

// PublicSurfaceOptions carries the shared cross-cutting inputs the public
// registry is allowed to pass into invite composition.
type PublicSurfaceOptions struct {
	RequestMeta requestmeta.SchemePolicy
	Principal   requestresolver.PrincipalResolver
}

// Compose builds the production invite module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	return New(Config{
		Gateway:       invitegateway.NewGRPCGateway(config.InviteClient, config.AuthClient),
		RequestMeta:   config.RequestMeta,
		Principal:     config.Principal,
		DashboardSync: dashboardsync.New(config.UserHubControlClient, config.GameEventClient, nil),
	})
}

// ComposePublic composes the public invite surface from the invite-owned
// dependency bundle plus request-scoped cross-cutting options.
func ComposePublic(options PublicSurfaceOptions, deps Dependencies) module.Module {
	return Compose(CompositionConfig{
		RequestMeta:          options.RequestMeta,
		Principal:            options.Principal,
		InviteClient:         deps.InviteClient,
		AuthClient:           deps.AuthClient,
		UserHubControlClient: deps.UserHubControlClient,
		GameEventClient:      deps.GameEventClient,
	})
}
