package modules

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/invite"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
)

// defaultPublicModules returns stable public web modules.
func defaultPublicModules(deps Dependencies, principal requestresolver.PrincipalResolver, opts PublicModuleOptions) []module.Module {
	publicModules := publicauth.ComposeSurfaceSet(publicauth.SurfaceSetConfig{
		AuthClient:  deps.PublicAuth.AuthClient,
		Principal:   principal,
		RequestMeta: opts.RequestSchemePolicy,
		AuthBaseURL: deps.PublicAuth.AuthBaseURL,
	})
	publicModules = append(publicModules,
		discovery.Compose(discovery.CompositionConfig{
			DiscoveryClient: deps.Discovery.DiscoveryClient,
		}),
		profile.Compose(profile.CompositionConfig{
			AuthClient:   deps.Profile.AuthClient,
			SocialClient: deps.Profile.SocialClient,
			AssetBaseURL: deps.AssetBaseURL,
			Principal:    principal,
		}),
		invite.ComposePublic(invite.PublicSurfaceOptions{
			RequestMeta: opts.RequestSchemePolicy,
			Principal:   principal,
		}, deps.Invite),
	)
	return publicModules
}
