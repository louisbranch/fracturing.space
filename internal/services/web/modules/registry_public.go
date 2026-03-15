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
	if discoveryModule, ok := discovery.ComposePublic(discovery.PublicSurfaceOptions{}, deps.Discovery); ok {
		publicModules = append(publicModules, discoveryModule)
	}
	if profileModule, ok := profile.ComposePublic(profile.PublicSurfaceOptions{
		AssetBaseURL: deps.AssetBaseURL,
		Principal:    requestPrincipal,
	}, deps.Profile); ok {
		publicModules = append(publicModules, profileModule)
	}
	if inviteModule, ok := invite.ComposePublic(invite.PublicSurfaceOptions{
		RequestMeta:   opts.RequestSchemePolicy,
		Principal:     requestPrincipal,
		DashboardSync: opts.DashboardSync,
	}, deps.Invite); ok {
		publicModules = append(publicModules, inviteModule)
	}
	return publicModules
}
