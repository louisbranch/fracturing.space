package modules

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	profilegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	publicauthgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/gateway"
)

// defaultPublicModules returns stable public web modules.
func defaultPublicModules(deps Dependencies, res ModuleResolvers, opts PublicModuleOptions) []module.Module {
	authGateway := publicauthgateway.NewGRPCGateway(deps.PublicAuth.AuthClient)
	discoveryGateway := discovery.NewGRPCGateway(deps.Discovery.DiscoveryClient)
	return []module.Module{
		publicauth.New(publicauth.Config{
			Gateway:     authGateway,
			RequestMeta: opts.RequestSchemePolicy,
			AuthBaseURL: deps.PublicAuth.AuthBaseURL,
			Surface:     publicauth.SurfaceShell,
		}),
		publicauth.New(publicauth.Config{
			Gateway:     authGateway,
			RequestMeta: opts.RequestSchemePolicy,
			AuthBaseURL: deps.PublicAuth.AuthBaseURL,
			Surface:     publicauth.SurfacePasskeys,
		}),
		publicauth.New(publicauth.Config{
			Gateway:     authGateway,
			RequestMeta: opts.RequestSchemePolicy,
			AuthBaseURL: deps.PublicAuth.AuthBaseURL,
			Surface:     publicauth.SurfaceAuthRedirect,
		}),
		discovery.New(discovery.Config{Gateway: discoveryGateway}),
		profile.New(profile.Config{
			Gateway:         profilegateway.NewGRPCGateway(deps.Profile.AuthClient, deps.Profile.SocialClient),
			AssetBaseURL:    deps.AssetBaseURL,
			ResolveSignedIn: res.ResolveSignedIn,
		}),
	}
}
