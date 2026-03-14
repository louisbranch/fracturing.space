package modules

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/invite"
	invitegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	profilegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	publicauthgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
)

// defaultPublicModules returns stable public web modules.
func defaultPublicModules(deps Dependencies, res ModuleResolvers, opts PublicModuleOptions) []module.Module {
	authGateway := publicauthgateway.NewGRPCGateway(deps.PublicAuth.AuthClient)
	discoveryGateway := discovery.NewGRPCGateway(deps.Discovery.DiscoveryClient)
	dashboardSync := dashboardsync.New(deps.DashboardSync.UserHubControlClient, deps.DashboardSync.GameEventClient, nil)
	return []module.Module{
		publicauth.New(publicauth.Config{
			Gateway:         authGateway,
			ResolveSignedIn: res.ResolveSignedIn,
			RequestMeta:     opts.RequestSchemePolicy,
			AuthBaseURL:     deps.PublicAuth.AuthBaseURL,
			Surface:         publicauth.SurfaceShell,
		}),
		publicauth.New(publicauth.Config{
			Gateway:         authGateway,
			ResolveSignedIn: res.ResolveSignedIn,
			RequestMeta:     opts.RequestSchemePolicy,
			AuthBaseURL:     deps.PublicAuth.AuthBaseURL,
			Surface:         publicauth.SurfacePasskeys,
		}),
		publicauth.New(publicauth.Config{
			Gateway:         authGateway,
			ResolveSignedIn: res.ResolveSignedIn,
			RequestMeta:     opts.RequestSchemePolicy,
			AuthBaseURL:     deps.PublicAuth.AuthBaseURL,
			Surface:         publicauth.SurfaceAuthRedirect,
		}),
		discovery.New(discovery.Config{Gateway: discoveryGateway}),
		profile.New(profile.Config{
			Gateway:         profilegateway.NewGRPCGateway(deps.Profile.AuthClient, deps.Profile.SocialClient),
			AssetBaseURL:    deps.AssetBaseURL,
			ResolveSignedIn: res.ResolveSignedIn,
		}),
		invite.New(invite.Config{
			Gateway:       invitegateway.NewGRPCGateway(deps.Campaigns.InviteClient, deps.Campaigns.AuthClient),
			Base:          publichandler.NewBase(publichandler.WithResolveViewerSignedIn(res.ResolveSignedIn)),
			RequestMeta:   opts.RequestSchemePolicy,
			ResolveUserID: res.ResolveUserID,
			DashboardSync: dashboardSync,
		}),
	}
}
