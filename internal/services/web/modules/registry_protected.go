package modules

import (
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// defaultProtectedModules returns stable authenticated web modules.
func defaultProtectedModules(deps Dependencies, res ModuleResolvers, opts ProtectedModuleOptions) []module.Module {
	return buildProtectedModules(deps, res, opts)
}

// buildProtectedModules centralizes protected module ordering while keeping
// production wiring inside the owning area packages.
func buildProtectedModules(
	deps Dependencies,
	res ModuleResolvers,
	opts ProtectedModuleOptions,
) []module.Module {
	base := modulehandler.NewBase(res.ResolveUserID, res.ResolveLanguage, res.ResolveViewer)
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
	protected = append(protected, campaigns.Compose(campaigns.CompositionConfig{
		Base:                     base,
		ChatFallbackPort:         opts.ChatFallbackPort,
		DashboardSync:            dashboardSync,
		AssetBaseURL:             deps.AssetBaseURL,
		CampaignClient:           deps.Campaigns.CampaignClient,
		CommunicationClient:      deps.Campaigns.CommunicationClient,
		AgentClient:              deps.Campaigns.AgentClient,
		ParticipantClient:        deps.Campaigns.ParticipantClient,
		CharacterClient:          deps.Campaigns.CharacterClient,
		DaggerheartContentClient: deps.Campaigns.DaggerheartContentClient,
		DaggerheartAssetClient:   deps.Campaigns.DaggerheartAssetClient,
		SessionClient:            deps.Campaigns.SessionClient,
		InviteClient:             deps.Campaigns.InviteClient,
		SocialClient:             deps.Campaigns.SocialClient,
		AuthClient:               deps.Campaigns.AuthClient,
		AuthorizationClient:      deps.Campaigns.AuthorizationClient,
	}))
	return protected
}
