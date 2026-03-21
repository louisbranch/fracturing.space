package publicauth

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	publicauthapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/app"
	publicauthgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// CompositionConfig owns the startup wiring required to construct the
// production publicauth module without leaking gateway internals into the
// registry package.
type CompositionConfig struct {
	AuthClient  publicauthgateway.AuthClient
	Principal   principal.PrincipalResolver
	RequestMeta requestmeta.SchemePolicy
	AuthBaseURL string
}

// SurfaceSetConfig owns the startup wiring required to construct the stable
// publicauth surface set without leaking route-surface ordering into the
// registry package.
type SurfaceSetConfig struct {
	AuthClient  publicauthgateway.AuthClient
	Principal   principal.PrincipalResolver
	RequestMeta requestmeta.SchemePolicy
	AuthBaseURL string
}

// compose builds one production publicauth module from shared startup
// composition config and applies the provided surface constructor.
func compose(config CompositionConfig, build func(Config) Module) module.Module {
	gateway := publicauthgateway.NewGRPCGateway(config.AuthClient)
	moduleConfig := Config{
		PageService:    publicauthapp.NewPageService(config.AuthBaseURL),
		SessionService: publicauthapp.NewSessionService(gateway, config.AuthBaseURL),
		PasskeyService: publicauthapp.NewPasskeyService(gateway),
		Recovery:       publicauthapp.NewRecoveryService(gateway),
		Principal:      config.Principal,
		RequestMeta:    config.RequestMeta,
	}
	return build(moduleConfig)
}

// ComposeShell builds the shell/public routes module.
func ComposeShell(config CompositionConfig) module.Module {
	return compose(config, NewShell)
}

// ComposePasskeys builds the passkey routes module.
func ComposePasskeys(config CompositionConfig) module.Module {
	return compose(config, NewPasskeys)
}

// ComposeAuthRedirect builds the auth-redirect routes module.
func ComposeAuthRedirect(config CompositionConfig) module.Module {
	return compose(config, NewAuthRedirect)
}

// newSharedSurfaceConfig builds the shared publicauth service bundle used by
// all route-owner modules in the stable surface set.
func newSharedSurfaceConfig(config SurfaceSetConfig) Config {
	gateway := publicauthgateway.NewGRPCGateway(config.AuthClient)
	return Config{
		PageService:    publicauthapp.NewPageService(config.AuthBaseURL),
		SessionService: publicauthapp.NewSessionService(gateway, config.AuthBaseURL),
		PasskeyService: publicauthapp.NewPasskeyService(gateway),
		Recovery:       publicauthapp.NewRecoveryService(gateway),
		Principal:      config.Principal,
		RequestMeta:    config.RequestMeta,
	}
}

// ComposeSurfaceSet builds the stable publicauth module set in area-owned
// order so the central registry only declares public module ordering.
func ComposeSurfaceSet(config SurfaceSetConfig) []module.Module {
	moduleConfig := newSharedSurfaceConfig(config)
	return []module.Module{
		NewShell(moduleConfig),
		NewPasskeys(moduleConfig),
		NewAuthRedirect(moduleConfig),
	}
}
