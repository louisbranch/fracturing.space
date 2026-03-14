package modules

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
)

// defaultProtectedModules returns stable authenticated web modules.
func defaultProtectedModules(deps Dependencies, principal requestresolver.PrincipalResolver, opts ProtectedModuleOptions) []module.Module {
	return buildProtectedModules(deps, principal, opts)
}

// buildProtectedModules centralizes protected module ordering while keeping
// production wiring inside the owning area packages.
func buildProtectedModules(
	deps Dependencies,
	principal requestresolver.PrincipalResolver,
	opts ProtectedModuleOptions,
) []module.Module {
	base := modulehandler.NewBaseFromPrincipal(principal)
	dashboardSync := dashboardsync.New(deps.DashboardSync.UserHubControlClient, deps.DashboardSync.GameEventClient, nil)

	protected := []module.Module{
		dashboard.Compose(dashboard.CompositionConfig{
			Base:          base,
			UserHubClient: deps.Dashboard.UserHubClient,
			StatusClient:  deps.Dashboard.StatusClient,
		}),
		settings.Compose(settings.CompositionConfig{
			Base:             base,
			FlashMeta:        opts.RequestSchemePolicy,
			DashboardSync:    dashboardSync,
			SocialClient:     deps.Settings.SocialClient,
			AccountClient:    deps.Settings.AccountClient,
			PasskeyClient:    deps.Settings.PasskeyClient,
			CredentialClient: deps.Settings.CredentialClient,
			AgentClient:      deps.Settings.AgentClient,
		}),
	}
	if deps.Notifications.NotificationClient != nil {
		protected = append(protected, notifications.Compose(notifications.CompositionConfig{
			Base:               base,
			NotificationClient: deps.Notifications.NotificationClient,
		}))
	}
	if campaignsModule, ok := campaigns.ComposeProtected(campaigns.ProtectedSurfaceOptions{
		Base:             base,
		ChatFallbackPort: opts.ChatFallbackPort,
		DashboardSync:    dashboardSync,
		AssetBaseURL:     deps.AssetBaseURL,
	}, deps.Campaigns); ok {
		protected = append(protected, campaignsModule)
	}
	return protected
}
