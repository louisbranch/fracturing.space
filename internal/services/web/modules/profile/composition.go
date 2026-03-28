package profile

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	profileapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/app"
	profilegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// Compose builds the profile module from the exact startup dependencies the
// area owns.
func Compose(
	authClient profilegateway.AuthClient,
	socialClient profilegateway.SocialClient,
	assetBaseURL string,
	principal principal.PrincipalResolver,
) module.Module {
	gateway := profilegateway.NewGRPCGateway(authClient, socialClient)
	return New(Config{
		Service:      profileapp.NewService(gateway),
		AssetBaseURL: assetBaseURL,
		Principal:    principal,
	})
}
