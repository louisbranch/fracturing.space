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

func serviceConfigsWithGateway(gateway testGatewayBundle) serviceConfigs {
	participantRead := campaignapp.ParticipantReadServiceConfig{
		Read:               gateway,
		Workspace:          gateway,
		BatchAuthorization: gateway,
	}
	authorization := campaignapp.AuthorizationGateway(gateway)
	return serviceConfigs{
		Page: pageServiceConfig{
			Workspace: campaignapp.WorkspaceServiceConfig{
				Read: gateway,
			},
			SessionRead: campaignapp.SessionReadServiceConfig{
				Read: gateway,
			},
			Authorization: authorization,
		},
		Catalog: catalogServiceConfig{
			Catalog: campaignapp.CatalogServiceConfig{
				Read:     gateway,
				Mutation: gateway,
			},
		},
		Starter: starterServiceConfig{
			Starter: campaignapp.StarterServiceConfig{
				Gateway: gateway,
			},
		},
		Overview: overviewServiceConfig{
			AutomationRead: campaignapp.AutomationReadServiceConfig{
				Participants: gateway,
				Read:         gateway,
			},
			AutomationMutation: campaignapp.AutomationMutationServiceConfig{
				Participants: gateway,
				Mutation:     gateway,
			},
			Configuration: campaignapp.ConfigurationServiceConfig{
				Workspace: gateway,
				Mutation:  gateway,
			},
			Authorization: authorization,
		},
		Participants: participantServiceConfig{
			Read: participantRead,
			Mutation: campaignapp.ParticipantMutationServiceConfig{
				Read:      gateway,
				Mutation:  gateway,
				Workspace: gateway,
			},
			Authorization: authorization,
		},
		Characters: characterServiceConfig{
			Read: campaignapp.CharacterReadServiceConfig{
				Read:               gateway,
				BatchAuthorization: gateway,
			},
			Control: campaignapp.CharacterControlServiceConfig{
				Read:         gateway,
				Mutation:     gateway,
				Participants: gateway,
				Sessions:     gateway,
			},
			Mutation: campaignapp.CharacterMutationServiceConfig{
				Mutation: gateway,
				Sessions: gateway,
			},
			Creation: campaignapp.CharacterCreationServiceConfig{
				Read:     gateway,
				Mutation: gateway,
			},
			Authorization: authorization,
		},
		Sessions: sessionServiceConfig{
			Mutation: campaignapp.SessionMutationServiceConfig{
				Mutation: gateway,
			},
		},
		Invites: inviteServiceConfig{
			Read: campaignapp.InviteReadServiceConfig{
				Read: gateway,
			},
			Mutation: campaignapp.InviteMutationServiceConfig{
				Mutation: gateway,
			},
			ParticipantRead: participantRead,
			Authorization:   authorization,
		},
	}
}

func configWithGateway(gateway testGatewayBundle, base modulehandler.Base, workflows campaignworkflow.Registry) Config {
	return Config{
		Services:         newHandlerServices(serviceConfigsWithGateway(gateway)),
		Base:             base,
		PlayFallbackPort: "",
		PlayLaunchGrant:  fakePlayLaunchGrantConfig(),
		Workflows:        workflows,
	}
}

func serviceConfigsWithGRPCDeps(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) serviceConfigs {
	return newServiceConfigsFromGRPCDeps(deps, assetBaseURL)
}

func configWithGRPCDeps(deps campaigngateway.GRPCGatewayDeps, base modulehandler.Base, workflows campaignworkflow.Registry) Config {
	return Config{
		Services:         newHandlerServices(serviceConfigsWithGRPCDeps(deps, "")),
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
