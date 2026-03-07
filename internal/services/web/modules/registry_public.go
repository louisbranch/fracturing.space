package modules

import (
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	profilegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/gateway"
	publicauthgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/surfaces/authredirect"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/surfaces/passkeys"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/surfaces/shell"
)

// defaultPublicModules returns stable public web modules.
func defaultPublicModules(deps Dependencies, res ModuleResolvers, opts PublicModuleOptions) []Module {
	authGateway := publicauthgateway.NewGRPCGateway(deps.AuthClient)
	discoveryGateway := discovery.NewGRPCGateway(deps.ListingClient)
	return []Module{
		shell.NewWithGatewayAndPolicy(authGateway, opts.RequestSchemePolicy),
		passkeys.NewWithGatewayAndPolicy(authGateway, opts.RequestSchemePolicy),
		authredirect.NewWithGatewayAndPolicy(authGateway, opts.RequestSchemePolicy),
		discovery.NewWithGateway(discoveryGateway),
		profile.NewWithGateway(profilegateway.NewGRPCGateway(deps.ProfileSocialClient), deps.AssetBaseURL, res.ResolveSignedIn),
	}
}
