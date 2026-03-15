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

// Compose builds the production profile module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	gateway := profilegateway.NewGRPCGateway(config.AuthClient, config.SocialClient)
	return New(Config{
		Service:      profileapp.NewService(gateway),
		AssetBaseURL: config.AssetBaseURL,
		Principal:    config.Principal,
		Healthy:      profileapp.IsGatewayHealthy(gateway),
	})
}
