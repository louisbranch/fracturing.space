package modules

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// defaultProtectedModules returns stable authenticated web modules.
func defaultProtectedModules(deps Dependencies, requestPrincipal principal.PrincipalResolver, opts ProtectedModuleOptions) []module.Module {
	return buildProtectedModules(deps, requestPrincipal, opts)
}

// buildProtectedModules centralizes protected module ordering while keeping
// production wiring inside the owning area packages.
func buildProtectedModules(
	deps Dependencies,
	requestPrincipal principal.PrincipalResolver,
	opts ProtectedModuleOptions,
) []module.Module {
	base := modulehandler.NewBaseFromPrincipal(requestPrincipal)
	dashboardOptions := dashboard.ProtectedSurfaceOptions{
		Base:   base,
		Logger: opts.Logger,
	}
	settingsOptions := settings.ProtectedSurfaceOptions{
		Base:          base,
		FlashMeta:     opts.RequestSchemePolicy,
		DashboardSync: opts.DashboardSync,
	}
	notificationsOptions := notifications.ProtectedSurfaceOptions{
		Base: base,
	}
	campaignsOptions := campaigns.ProtectedSurfaceOptions{
		Base:             base,
		PlayFallbackPort: opts.PlayFallbackPort,
		PlayLaunchGrant:  opts.PlayLaunchGrant,
		RequestMeta:      opts.RequestSchemePolicy,
		DashboardSync:    opts.DashboardSync,
		AssetBaseURL:     deps.AssetBaseURL,
	}

	protected := []module.Module{
		dashboard.ComposeProtected(dashboardOptions, deps.Dashboard),
		settings.ComposeProtected(settingsOptions, deps.Settings),
	}
	if notificationsModule, ok := notifications.ComposeProtected(notificationsOptions, deps.Notifications); ok {
		protected = append(protected, notificationsModule)
	}
	if campaignsModule, ok := campaigns.ComposeProtected(campaignsOptions, deps.Campaigns); ok {
		protected = append(protected, campaignsModule)
	}
	return protected
}
