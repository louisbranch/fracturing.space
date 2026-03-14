package modules

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
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
func defaultProtectedModules(deps Dependencies, res ModuleResolvers, opts ProtectedModuleOptions) []module.Module {
	return buildProtectedModules(deps, res, opts)
}

// buildProtectedModules centralizes this web behavior in one helper seam.
func buildProtectedModules(
	deps Dependencies,
	res ModuleResolvers,
	opts ProtectedModuleOptions,
) []module.Module {
	base := modulehandler.NewBase(res.ResolveUserID, res.ResolveLanguage, res.ResolveViewer)
	dashboardSync := dashboardsync.New(deps.DashboardSync.UserHubControlClient, deps.DashboardSync.GameEventClient, nil)
	campaignMod := newCampaignModule(deps, base, opts.ChatFallbackPort, dashboardSync, defaultCampaignWorkflows(deps))
	settingsMod := settings.New(settings.Config{
		Gateway:       settingsgateway.NewGRPCGateway(deps.Settings.SocialClient, deps.Settings.AccountClient, deps.Settings.PasskeyClient, deps.Settings.CredentialClient, deps.Settings.AgentClient),
		Base:          base,
		FlashMeta:     opts.RequestSchemePolicy,
		DashboardSync: dashboardSync,
	})
	dashGw := dashboardgateway.NewGRPCGateway(deps.Dashboard.UserHubClient)
	dashMod := dashboard.New(dashboard.Config{
		Gateway:        dashGw,
		Base:           base,
		HealthProvider: statusHealthProvider(deps.Dashboard.StatusClient),
	})
	protected := []module.Module{dashMod, settingsMod}
	if deps.Notifications.NotificationClient != nil {
		protected = append(protected, notifications.New(notifications.Config{
			Gateway: notificationsgateway.NewGRPCGateway(deps.Notifications.NotificationClient),
			Base:    base,
		}))
	}
	protected = append(protected, campaignMod)
	return protected
}

// defaultCampaignWorkflows returns the production workflow implementations
// keyed by canonical game-system identifiers.
func defaultCampaignWorkflows(deps Dependencies) campaignworkflow.Registry {
	return campaignworkflow.Registry{
		campaignapp.GameSystemDaggerheart: daggerheart.New(deps.AssetBaseURL),
	}
}

// newCampaignModule returns the campaigns module with stable route ownership.
func newCampaignModule(
	deps Dependencies,
	base modulehandler.Base,
	chatFallbackPort string,
	dashboardSync campaigns.DashboardSync,
	workflows campaignworkflow.Registry,
) module.Module {
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
	return func(ctx context.Context) []dashboardapp.ServiceHealthEntry {
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
		entries := make([]dashboardapp.ServiceHealthEntry, 0, len(services))
		for _, svc := range services {
			if svc == nil {
				continue
			}
			entries = append(entries, dashboardapp.ServiceHealthEntry{
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
func newCampaignGateway(deps Dependencies) campaignapp.CampaignGateway {
	return campaigngateway.NewGRPCGateway(campaigngateway.GRPCGatewayDeps{
		Read: campaigngateway.GRPCGatewayReadDeps{
			Campaign:           deps.Campaigns.CampaignClient,
			Communication:      deps.Campaigns.CommunicationClient,
			Agent:              deps.Campaigns.AgentClient,
			Participant:        deps.Campaigns.ParticipantClient,
			Character:          deps.Campaigns.CharacterClient,
			DaggerheartContent: deps.Campaigns.DaggerheartContentClient,
			DaggerheartAsset:   deps.Campaigns.DaggerheartAssetClient,
			Session:            deps.Campaigns.SessionClient,
			Invite:             deps.Campaigns.InviteClient,
		},
		Mutation: campaigngateway.GRPCGatewayMutationDeps{
			Campaign:    deps.Campaigns.CampaignClient,
			Participant: deps.Campaigns.ParticipantClient,
			Character:   deps.Campaigns.CharacterClient,
			Session:     deps.Campaigns.SessionClient,
			Invite:      deps.Campaigns.InviteClient,
			Auth:        deps.Campaigns.AuthClient,
		},
		Authorization: campaigngateway.GRPCGatewayAuthorizationDeps{
			Client: deps.Campaigns.AuthorizationClient,
		},
		AssetBaseURL: deps.AssetBaseURL,
	})
}
