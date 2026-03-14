package modules

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard"
	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	dashboardgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications"
	notificationsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/settings"
	settingsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// defaultProtectedModules returns stable authenticated web modules.
func defaultProtectedModules(deps Dependencies, res ModuleResolvers, opts ProtectedModuleOptions) []Module {
	return buildProtectedModules(deps, res, opts)
}

// buildProtectedModules centralizes this web behavior in one helper seam.
func buildProtectedModules(
	deps Dependencies,
	res ModuleResolvers,
	opts ProtectedModuleOptions,
) []Module {
	base := modulehandler.NewBase(res.ResolveUserID, res.ResolveLanguage, res.ResolveViewer)
	dashboardSync := dashboardsync.New(deps.DashboardSync.UserHubControlClient, deps.DashboardSync.GameEventClient, nil)
	campaignMod := newCampaignModule(deps, base, opts.ChatFallbackPort, dashboardSync, defaultCampaignWorkflows(deps))
	settingsMod := settings.New(settings.Config{
		Gateway:       settingsgateway.NewGRPCGateway(deps.Settings.SocialClient, deps.Settings.AccountClient, deps.Settings.PasskeyClient, deps.Settings.CredentialClient, deps.Settings.AgentClient),
		Base:          base,
		FlashMeta:     opts.RequestSchemePolicy,
		DashboardSync: dashboardSync,
	})
	notifMod := notifications.New(notifications.Config{
		Gateway: notificationsgateway.NewGRPCGateway(deps.Notifications.NotificationClient),
		Base:    base,
	})

	dashGw := dashboardgateway.NewGRPCGateway(deps.Dashboard.UserHubClient)
	dashMod := dashboard.New(dashboard.Config{
		Gateway:        dashGw,
		Base:           base,
		HealthProvider: statusHealthProvider(deps.Dashboard.StatusClient),
	})
	return []Module{dashMod, settingsMod, notifMod, campaignMod}
}

// defaultCampaignWorkflows returns the production workflow implementations
// keyed by canonical game-system identifiers.
func defaultCampaignWorkflows(deps Dependencies) map[campaigns.GameSystem]campaigns.CharacterCreationWorkflow {
	return map[campaigns.GameSystem]campaigns.CharacterCreationWorkflow{
		campaigns.GameSystemDaggerheart: daggerheart.New(deps.AssetBaseURL),
	}
}

// newCampaignModule returns the campaigns module with stable route ownership.
func newCampaignModule(
	deps Dependencies,
	base modulehandler.Base,
	chatFallbackPort string,
	dashboardSync campaigns.DashboardSync,
	workflows map[campaigns.GameSystem]campaigns.CharacterCreationWorkflow,
) Module {
	return campaigns.New(campaigns.Config{
		Gateway:          newCampaignGateway(deps),
		Base:             base,
		ChatFallbackPort: chatFallbackPort,
		Workflows:        workflows,
		DashboardSync:    dashboardSync,
	})
}

// statusHealthTimeout caps a per-request status service query.
const statusHealthTimeout = 3 * time.Second

// statusHealthProvider returns a HealthProvider that queries the status service
// on each dashboard load. Returns nil when no status client is available.
func statusHealthProvider(client statusv1.StatusServiceClient) dashboardapp.HealthProvider {
	if client == nil {
		return nil
	}
	return func(ctx context.Context) []dashboard.ServiceHealthEntry {
		ctx, cancel := context.WithTimeout(ctx, statusHealthTimeout)
		defer cancel()
		resp, err := client.GetSystemStatus(ctx, &statusv1.GetSystemStatusRequest{})
		if err != nil {
			log.Printf("web: status service health query failed: %v", err)
			return nil
		}
		services := resp.GetServices()
		if len(services) == 0 {
			return nil
		}
		entries := make([]dashboard.ServiceHealthEntry, 0, len(services))
		for _, svc := range services {
			if svc == nil {
				continue
			}
			entries = append(entries, dashboard.ServiceHealthEntry{
				Label:     capitalizeLabel(strings.TrimSpace(svc.GetService())),
				Available: svc.GetAggregateStatus() == statusv1.CapabilityStatus_CAPABILITY_STATUS_OPERATIONAL,
			})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Label < entries[j].Label
		})
		return entries
	}
}

// newCampaignGateway builds package wiring for this web seam.
func newCampaignGateway(deps Dependencies) campaigns.CampaignGateway {
	return campaigns.NewGRPCGateway(campaigns.GRPCGatewayDeps{
		CampaignClient:           deps.Campaigns.CampaignClient,
		AgentClient:              deps.Campaigns.AgentClient,
		ParticipantClient:        deps.Campaigns.ParticipantClient,
		CharacterClient:          deps.Campaigns.CharacterClient,
		DaggerheartContentClient: deps.Campaigns.DaggerheartContentClient,
		DaggerheartAssetClient:   deps.Campaigns.DaggerheartAssetClient,
		SessionClient:            deps.Campaigns.SessionClient,
		InviteClient:             deps.Campaigns.InviteClient,
		AuthorizationClient:      deps.Campaigns.AuthorizationClient,
		AssetBaseURL:             deps.AssetBaseURL,
	})
}
