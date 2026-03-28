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
		publicModules = append(publicModules, discovery.Compose(
			deps.Discovery.DiscoveryClient,
			opts.Logger,
		))
	}
	if deps.Profile.AuthClient != nil {
		publicModules = append(publicModules, profile.Compose(
			deps.Profile.AuthClient,
			deps.Profile.SocialClient,
			deps.AssetBaseURL,
			requestPrincipal,
		))
	}
	if deps.Invite.InviteClient != nil && deps.Invite.AuthClient != nil {
		publicModules = append(publicModules, invite.Compose(
			deps.Invite.InviteClient,
			deps.Invite.AuthClient,
			opts.RequestSchemePolicy,
			requestPrincipal,
			opts.DashboardSync,
		))
	}
	return publicModules
}
