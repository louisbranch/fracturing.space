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
	Surface     Surface
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

// Compose builds the production publicauth module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	gateway := publicauthgateway.NewGRPCGateway(config.AuthClient)
	return New(Config{
		PageService:    publicauthapp.NewPageService(config.AuthBaseURL),
		SessionService: publicauthapp.NewSessionService(gateway, config.AuthBaseURL),
		PasskeyService: publicauthapp.NewPasskeyService(gateway, config.AuthBaseURL),
		Recovery:       publicauthapp.NewRecoveryService(gateway, config.AuthBaseURL),
		Principal:      config.Principal,
		RequestMeta:    config.RequestMeta,
		Surface:        config.Surface,
	})
}

// ComposeSurfaceSet builds the stable publicauth module set in area-owned
// order so the central registry only declares public module ordering.
func ComposeSurfaceSet(config SurfaceSetConfig) []module.Module {
	return []module.Module{
		Compose(CompositionConfig{
			AuthClient:  config.AuthClient,
			Principal:   config.Principal,
			RequestMeta: config.RequestMeta,
			AuthBaseURL: config.AuthBaseURL,
			Surface:     SurfaceShell,
		}),
		Compose(CompositionConfig{
			AuthClient:  config.AuthClient,
			Principal:   config.Principal,
			RequestMeta: config.RequestMeta,
			AuthBaseURL: config.AuthBaseURL,
			Surface:     SurfacePasskeys,
		}),
		Compose(CompositionConfig{
			AuthClient:  config.AuthClient,
			Principal:   config.Principal,
			RequestMeta: config.RequestMeta,
			AuthBaseURL: config.AuthBaseURL,
			Surface:     SurfaceAuthRedirect,
		}),
	}
}
