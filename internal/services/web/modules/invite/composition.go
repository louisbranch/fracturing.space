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
	})
}

// ComposePublic composes the public invite surface when required dependencies are
// available. The registry can use this to hide optional invite routes instead of
// keeping a fail-closed fallback.
func ComposePublic(options PublicSurfaceOptions, deps Dependencies) (module.Module, bool) {
	if !deps.configured() {
		return nil, false
	}
	return Compose(newCompositionConfig(options, deps)), true
}

// newCompositionConfig projects startup dependencies and shared options into invite
// composition input.
func newCompositionConfig(options PublicSurfaceOptions, deps Dependencies) CompositionConfig {
	return CompositionConfig{
		RequestMeta:   options.RequestMeta,
		Principal:     options.Principal,
		DashboardSync: options.DashboardSync,
		InviteClient:  deps.InviteClient,
		AuthClient:    deps.AuthClient,
	}
}

// configured reports whether the invite dependency set has the clients required
// for production-safe mounting.
func (deps Dependencies) configured() bool {
	return deps.InviteClient != nil && deps.AuthClient != nil
}
