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
	settingsOptions := settings.ProtectedSurfaceOptions{
		Base:          base,
		FlashMeta:     opts.RequestSchemePolicy,
		DashboardSync: opts.DashboardSync,
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
		dashboard.Compose(
			deps.Dashboard.UserHubClient,
			deps.Dashboard.StatusClient,
			base,
			opts.Logger,
		),
		settings.ComposeProtected(settingsOptions, deps.Settings),
	}
	if deps.Notifications.NotificationClient != nil {
		protected = append(protected, notifications.Compose(
			deps.Notifications.NotificationClient,
			base,
		))
	}
	if campaignsModule, ok := campaigns.ComposeProtected(campaignsOptions, deps.Campaigns); ok {
		protected = append(protected, campaignsModule)
	}
	return protected
}
