package profile

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	profilegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// CompositionConfig owns the startup wiring required to construct the
// production profile module without leaking gateway internals into the
// registry package.
type CompositionConfig struct {
	AuthClient   profilegateway.AuthClient
	SocialClient profilegateway.SocialClient
	AssetBaseURL string
	Principal    principal.PrincipalResolver
}

// PublicSurfaceOptions carries shared cross-cutting inputs the public registry is
// allowed to pass into profile composition.
type PublicSurfaceOptions struct {
	AssetBaseURL string
	Principal    principal.PrincipalResolver
}

// Compose builds the production profile module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	gateway := profilegateway.NewGRPCGateway(config.AuthClient, config.SocialClient)
	return New(Config{
		Service:      profileapp.NewService(gateway),
		AssetBaseURL: config.AssetBaseURL,
		Principal:    config.Principal,
	})
}

// ComposePublic composes the profile public surface when required dependencies
// are present. The registry can use this to hide optional public routes when
// backend clients are missing.
func ComposePublic(options PublicSurfaceOptions, deps Dependencies) (module.Module, bool) {
	if !deps.configured() {
		return nil, false
	}
	return Compose(newCompositionConfig(options, deps)), true
}

// newCompositionConfig projects startup dependencies and shared options into
// profile composition input.
func newCompositionConfig(options PublicSurfaceOptions, deps Dependencies) CompositionConfig {
	return CompositionConfig{
		AuthClient:   deps.AuthClient,
		SocialClient: deps.SocialClient,
		AssetBaseURL: options.AssetBaseURL,
		Principal:    options.Principal,
	}
}

// configured reports whether the profile dependency set has the clients required
// for production-safe mounting.
func (deps Dependencies) configured() bool {
	return deps.AuthClient != nil
}
