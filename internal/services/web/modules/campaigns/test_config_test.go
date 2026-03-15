package campaigns

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

type testGatewayBundle interface {
	campaignapp.CampaignCatalogReadGateway
	campaignapp.CampaignStarterGateway
	campaignapp.CampaignWorkspaceReadGateway
	campaignapp.CampaignGameReadGateway
	campaignapp.CampaignParticipantReadGateway
	campaignapp.CampaignCharacterReadGateway
	campaignapp.CampaignSessionReadGateway
	campaignapp.CampaignInviteReadGateway
	campaignapp.CampaignAutomationReadGateway
	campaignapp.CampaignCatalogMutationGateway
	campaignapp.CampaignConfigurationMutationGateway
	campaignapp.CampaignAutomationMutationGateway
	campaignapp.CampaignCharacterControlMutationGateway
	campaignapp.CampaignCharacterMutationGateway
	campaignapp.CampaignParticipantMutationGateway
	campaignapp.CampaignSessionMutationGateway
	campaignapp.CampaignInviteMutationGateway
	campaignapp.AuthorizationGateway
	campaignapp.BatchAuthorizationGateway
	campaignapp.CharacterCreationReadGateway
	campaignapp.CharacterCreationMutationGateway
}

func serviceConfigWithGateway(gateway testGatewayBundle) campaignapp.ServiceConfig {
	return campaignapp.ServiceConfig{
		Catalog: campaignapp.CatalogServiceConfig{
			Read:     gateway,
			Mutation: gateway,
		},
		Starter: campaignapp.StarterServiceConfig{
			Gateway: gateway,
		},
		Workspace: campaignapp.WorkspaceServiceConfig{
			Read: gateway,
		},
		Game: campaignapp.GameServiceConfig{
			Read: gateway,
		},
		ParticipantRead: campaignapp.ParticipantReadServiceConfig{
			Read:               gateway,
			Workspace:          gateway,
			BatchAuthorization: gateway,
		},
		ParticipantMutation: campaignapp.ParticipantMutationServiceConfig{
			Read:      gateway,
			Mutation:  gateway,
			Workspace: gateway,
		},
		CharacterRead: campaignapp.CharacterReadServiceConfig{
			Read:               gateway,
			BatchAuthorization: gateway,
		},
		CharacterControl: campaignapp.CharacterControlServiceConfig{
			Read:         gateway,
			Mutation:     gateway,
			Participants: gateway,
			Sessions:     gateway,
		},
		CharacterMutation: campaignapp.CharacterMutationServiceConfig{
			Mutation: gateway,
			Sessions: gateway,
		},
		SessionRead: campaignapp.SessionReadServiceConfig{
			Read: gateway,
		},
		SessionMutation: campaignapp.SessionMutationServiceConfig{
			Mutation: gateway,
		},
		InviteRead: campaignapp.InviteReadServiceConfig{
			Read: gateway,
		},
		InviteMutation: campaignapp.InviteMutationServiceConfig{
			Mutation: gateway,
		},
		Configuration: campaignapp.ConfigurationServiceConfig{
			Workspace: gateway,
			Mutation:  gateway,
		},
		AutomationRead: campaignapp.AutomationReadServiceConfig{
			Participants: gateway,
			Read:         gateway,
		},
		AutomationMutation: campaignapp.AutomationMutationServiceConfig{
			Participants: gateway,
			Mutation:     gateway,
		},
		Creation: campaignapp.CharacterCreationServiceConfig{
			Read:     gateway,
			Mutation: gateway,
		},
		Authorization: gateway,
	}
}

func configWithGateway(gateway testGatewayBundle, base modulehandler.Base, workflows campaignworkflow.Registry) Config {
	return Config{
		Services:         newHandlerServices(serviceConfigWithGateway(gateway)),
		Base:             base,
		PlayFallbackPort: "",
		PlayLaunchGrant:  fakePlayLaunchGrantConfig(),
		Workflows:        workflows,
	}
}

func serviceConfigWithGRPCDeps(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.ServiceConfig {
	return newServiceConfigFromGRPCDeps(deps, assetBaseURL)
}

func configWithGRPCDeps(deps campaigngateway.GRPCGatewayDeps, base modulehandler.Base, workflows campaignworkflow.Registry) Config {
	return Config{
		Services:         newHandlerServices(serviceConfigWithGRPCDeps(deps, "")),
		Base:             base,
		PlayFallbackPort: "",
		PlayLaunchGrant:  fakePlayLaunchGrantConfig(),
		Workflows:        workflows,
	}
}

func configWithGatewayAndSync(
	gateway testGatewayBundle,
	base modulehandler.Base,
	workflows campaignworkflow.Registry,
	sync DashboardSync,
) Config {
	cfg := configWithGateway(gateway, base, workflows)
	cfg.DashboardSync = sync
	return cfg
}

func fakePlayLaunchGrantConfig() playlaunchgrant.Config {
	return playlaunchgrant.Config{
		Issuer:   "issuer-test",
		Audience: "audience-test",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      time.Minute,
		Now:      func() time.Time { return time.Date(2026, 3, 13, 16, 0, 0, 0, time.UTC) },
	}
}
