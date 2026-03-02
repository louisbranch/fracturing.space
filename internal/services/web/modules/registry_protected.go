package modules

import (
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	dashboardgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	notificationsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/profile"
	profilegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	settingsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// defaultProtectedModules returns stable authenticated web modules.
func defaultProtectedModules(deps Dependencies, res ModuleResolvers, opts ProtectedModuleOptions) []Module {
	modules, _ := buildProtectedModules(deps, res, opts)
	return modules
}

// buildProtectedModules centralizes this web behavior in one helper seam.
func buildProtectedModules(
	deps Dependencies,
	res ModuleResolvers,
	opts ProtectedModuleOptions,
) ([]Module, []dashboard.ServiceHealthEntry) {
	base := modulehandler.NewBase(res.ResolveUserID, res.ResolveLanguage, res.ResolveViewer)
	campaignMod := newCampaignModule(deps, base, opts.ChatFallbackPort)
	settingsMod := settings.New(settings.WithGateway(settingsgateway.NewGRPCGateway(deps.SettingsSocialClient, deps.AccountClient, deps.CredentialClient)), settings.WithBase(base), settings.WithSchemePolicy(opts.RequestSchemePolicy))
	notifMod := notifications.NewWithGateway(notificationsgateway.NewGRPCGateway(deps.NotificationClient), base)
	profileProbe := profile.NewWithGateway(profilegateway.NewGRPCGateway(deps.ProfileSocialClient), deps.AssetBaseURL, res.ResolveSignedIn)

	// Dashboard's own health is derived from a probe module — the dashboard
	// module is constructed last because it receives the complete health list.
	dashGw := dashboardgateway.NewGRPCGateway(deps.UserHubClient)
	dashProbe := dashboard.NewWithGateway(dashGw, base, nil)

	health := DeriveServiceHealth([]Module{
		profileProbe,
		settingsMod,
		notifMod,
		campaignMod,
		dashProbe,
	})

	dashMod := dashboard.NewWithGateway(dashGw, base, health)
	return []Module{dashMod, settingsMod, notifMod, campaignMod}, health
}

// defaultCampaignWorkflows returns the production workflow implementations
// keyed by their system label (lowercase).
func defaultCampaignWorkflows() map[string]campaigns.CharacterCreationWorkflow {
	return map[string]campaigns.CharacterCreationWorkflow{
		"daggerheart": daggerheart.New(),
	}
}

// newCampaignModule returns the campaigns module with stable route ownership.
func newCampaignModule(deps Dependencies, base modulehandler.Base, chatFallbackPort string) Module {
	return campaigns.NewStableWithGateway(newCampaignGateway(deps), base, chatFallbackPort, defaultCampaignWorkflows())
}

// newCampaignGateway builds package wiring for this web seam.
func newCampaignGateway(deps Dependencies) campaigns.CampaignGateway {
	return campaigns.NewGRPCGateway(campaigns.GRPCGatewayDeps{
		CampaignClient:           deps.CampaignClient,
		ParticipantClient:        deps.ParticipantClient,
		CharacterClient:          deps.CharacterClient,
		DaggerheartContentClient: deps.DaggerheartContentClient,
		SessionClient:            deps.SessionClient,
		InviteClient:             deps.InviteClient,
		AuthorizationClient:      deps.AuthorizationClient,
		AssetBaseURL:             deps.AssetBaseURL,
	})
}
