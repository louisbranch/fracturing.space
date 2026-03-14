package profile

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	profilegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
)

// CompositionConfig owns the startup wiring required to construct the
// production profile module without leaking gateway internals into the
// registry package.
type CompositionConfig struct {
	AuthClient   profilegateway.AuthClient
	SocialClient profilegateway.SocialClient
	AssetBaseURL string
	Principal    requestresolver.PrincipalResolver
}

// Compose builds the production profile module from area-owned startup
// dependencies.
func Compose(config CompositionConfig) module.Module {
	return New(Config{
		Gateway:      profilegateway.NewGRPCGateway(config.AuthClient, config.SocialClient),
		AssetBaseURL: config.AssetBaseURL,
		Principal:    config.Principal,
	})
}
