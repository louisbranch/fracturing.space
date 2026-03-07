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
	authGateway := publicauthgateway.NewGRPCGateway(deps.PublicAuth.AuthClient)
	discoveryGateway := discovery.NewGRPCGateway(deps.Discovery.DiscoveryClient)
	return []Module{
		shell.New(shell.Config{
			Gateway:     authGateway,
			RequestMeta: opts.RequestSchemePolicy,
		}),
		passkeys.New(passkeys.Config{
			Gateway:     authGateway,
			RequestMeta: opts.RequestSchemePolicy,
		}),
		authredirect.New(authredirect.Config{
			Gateway:     authGateway,
			RequestMeta: opts.RequestSchemePolicy,
		}),
		discovery.New(discovery.Config{Gateway: discoveryGateway}),
		profile.New(profile.Config{
			Gateway:         profilegateway.NewGRPCGateway(deps.Profile.SocialClient),
			AssetBaseURL:    deps.AssetBaseURL,
			ResolveSignedIn: res.ResolveSignedIn,
		}),
	}
}
