package modules

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/invite"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// defaultPublicModules returns stable public web modules.
func defaultPublicModules(deps Dependencies, requestPrincipal principal.PrincipalResolver, opts PublicModuleOptions) []module.Module {
	publicModules := publicauth.ComposeSurfaceSet(publicauth.SurfaceSetConfig{
		AuthClient:  deps.PublicAuth.AuthClient,
		Principal:   requestPrincipal,
		RequestMeta: opts.RequestSchemePolicy,
		AuthBaseURL: deps.PublicAuth.AuthBaseURL,
	})
	if deps.Discovery.DiscoveryClient != nil {
		publicModules = append(publicModules, discovery.Compose(discovery.CompositionConfig{
			Client: deps.Discovery.DiscoveryClient,
			Logger: opts.Logger,
		}))
	}
	if deps.Profile.AuthClient != nil {
		publicModules = append(publicModules, profile.Compose(profile.CompositionConfig{
			AuthClient:   deps.Profile.AuthClient,
			SocialClient: deps.Profile.SocialClient,
			AssetBaseURL: deps.AssetBaseURL,
			Principal:    requestPrincipal,
		}))
	}
	if deps.Invite.InviteClient != nil && deps.Invite.AuthClient != nil {
		publicModules = append(publicModules, invite.Compose(invite.CompositionConfig{
			InviteClient:  deps.Invite.InviteClient,
			AuthClient:    deps.Invite.AuthClient,
			RequestMeta:   opts.RequestSchemePolicy,
			Principal:     requestPrincipal,
			DashboardSync: opts.DashboardSync,
		}))
	}
	return publicModules
}
