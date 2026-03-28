package campaigns

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigncharacters "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/characters"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
	campaigninvites "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/invites"
	campaignoverview "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/overview"
	campaignparticipants "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/participants"
	campaignsessions "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/sessions"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

type serviceConfigs struct {
	Page         campaigndetail.PageServiceConfig
	Catalog      catalogServiceConfig
	Starter      starterServiceConfig
	Overview     campaignoverview.ServiceConfig
	Participants campaignparticipants.ServiceConfig
	Characters   campaigncharacters.ServiceConfig
	Sessions     campaignsessions.ServiceConfig
	Invites      campaigninvites.ServiceConfig
}

func newTestCampaignSystems(workflows ...campaignworkflow.Registry) campaignSystemRegistry {
	return newCampaignSystemsFromWorkflows(workflows...)
}

func newHandlerServices(config serviceConfigs, workflows ...campaignworkflow.Registry) handlerServices {
	systems := newCampaignSystemsFromWorkflows(workflows...)
	page, err := campaigndetail.NewPageServices(config.Page)
	if err != nil {
		panic(err)
	}
	catalog, err := newCatalogHandlerServices(config.Catalog)
	if err != nil {
		panic(err)
	}
	starter, err := newStarterHandlerServices(config.Starter)
	if err != nil {
		panic(err)
	}
	overview, err := campaignoverview.NewHandlerServices(config.Overview)
	if err != nil {
		panic(err)
	}
	participants, err := campaignparticipants.NewHandlerServices(config.Participants)
	if err != nil {
		panic(err)
	}
	characters, err := campaigncharacters.NewHandlerServices(config.Characters, systems.workflowRegistry())
	if err != nil {
		panic(err)
	}
	sessions, err := campaignsessions.NewHandlerServices(config.Sessions)
	if err != nil {
		panic(err)
	}
	invites, err := campaigninvites.NewHandlerServices(config.Invites)
	if err != nil {
		panic(err)
	}
	return handlerServices{
		Page:         page,
		Catalog:      catalog,
		Starter:      starter,
		Overview:     overview,
		Participants: participants,
		Characters:   characters,
		Sessions:     sessions,
		Invites:      invites,
	}
}

func newHandlersFromConfig(
	config serviceConfigs,
	base modulehandler.Base,
	sync campaigndetail.DashboardSync,
	workflows ...campaignworkflow.Registry,
) handlers {
	handlerSet, err := newHandlers(handlersConfig{
		Services: newHandlerServices(config, workflows...),
		Base:     base,
		Sync:     sync,
		Systems:  newCampaignSystemsFromWorkflows(workflows...),
	})
	if err != nil {
		panic(err)
	}
	return handlerSet
}

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
	campaignapp.CampaignCharacterOwnershipMutationGateway
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
		Page: campaigndetail.PageServiceConfig{
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
		Overview: campaignoverview.ServiceConfig{
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
		Participants: campaignparticipants.ServiceConfig{
			Read: participantRead,
			Mutation: campaignapp.ParticipantMutationServiceConfig{
				Read:      gateway,
				Mutation:  gateway,
				Workspace: gateway,
			},
			Authorization: authorization,
		},
		Characters: campaigncharacters.ServiceConfig{
			Read: campaignapp.CharacterReadServiceConfig{
				Read:               gateway,
				BatchAuthorization: gateway,
			},
			Ownership: campaignapp.CharacterOwnershipServiceConfig{
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
		Sessions: campaignsessions.ServiceConfig{
			Characters: campaignapp.CharacterReadServiceConfig{
				Read:               gateway,
				BatchAuthorization: gateway,
			},
			Mutation: campaignapp.SessionMutationServiceConfig{
				Mutation: gateway,
			},
			Participants:  participantRead,
			Authorization: authorization,
		},
		Invites: campaigninvites.ServiceConfig{
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
		Services:         newHandlerServices(serviceConfigsWithGateway(gateway), workflows),
		Base:             base,
		PlayFallbackPort: "",
		PlayLaunchGrant:  fakePlayLaunchGrantConfig(),
		Systems:          newTestCampaignSystems(workflows),
	}
}

func serviceConfigsWithGRPCDeps(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) serviceConfigs {
	authorization := campaigngateway.NewAuthorizationGateway(deps.Page.Authorization)
	participantRead := campaignapp.ParticipantReadServiceConfig{
		Read:               campaigngateway.NewParticipantReadGateway(deps.Participants.Read, assetBaseURL),
		Workspace:          campaigngateway.NewWorkspaceReadGateway(deps.Participants.Workspace, assetBaseURL),
		BatchAuthorization: campaigngateway.NewBatchAuthorizationGateway(deps.Participants.Authorization),
	}
	return serviceConfigs{
		Page: campaigndetail.PageServiceConfig{
			Workspace: campaignapp.WorkspaceServiceConfig{
				Read: campaigngateway.NewWorkspaceReadGateway(deps.Page.Workspace, assetBaseURL),
			},
			SessionRead: campaignapp.SessionReadServiceConfig{
				Read: campaigngateway.NewSessionReadGateway(deps.Page.SessionRead),
			},
			Authorization: authorization,
		},
		Catalog: catalogServiceConfig{
			Catalog: campaignapp.CatalogServiceConfig{
				Read:     campaigngateway.NewCatalogReadGateway(deps.Catalog.Read, assetBaseURL),
				Mutation: campaigngateway.NewCatalogMutationGateway(deps.Catalog.Mutation),
			},
		},
		Starter: starterServiceConfig{
			Starter: campaignapp.StarterServiceConfig{
				Gateway: campaigngateway.NewStarterGateway(deps.Starter.Starter),
			},
		},
		Overview: campaignoverview.ServiceConfig{
			AutomationRead: campaignapp.AutomationReadServiceConfig{
				Participants: campaigngateway.NewParticipantReadGateway(deps.Overview.Participants, assetBaseURL),
				Read:         campaigngateway.NewAutomationReadGateway(deps.Overview.AutomationRead),
			},
			AutomationMutation: campaignapp.AutomationMutationServiceConfig{
				Participants: campaigngateway.NewParticipantReadGateway(deps.Overview.Participants, assetBaseURL),
				Mutation:     campaigngateway.NewAutomationMutationGateway(deps.Overview.AutomationMutation),
			},
			Configuration: campaignapp.ConfigurationServiceConfig{
				Workspace: campaigngateway.NewWorkspaceReadGateway(deps.Overview.Workspace, assetBaseURL),
				Mutation:  campaigngateway.NewConfigurationMutationGateway(deps.Overview.ConfigurationMutation),
			},
			Authorization: campaigngateway.NewAuthorizationGateway(deps.Overview.Authorization),
		},
		Participants: campaignparticipants.ServiceConfig{
			Read: participantRead,
			Mutation: campaignapp.ParticipantMutationServiceConfig{
				Read:      campaigngateway.NewParticipantReadGateway(deps.Participants.Read, assetBaseURL),
				Mutation:  campaigngateway.NewParticipantMutationGateway(deps.Participants.Mutation),
				Workspace: campaigngateway.NewWorkspaceReadGateway(deps.Participants.Workspace, assetBaseURL),
			},
			Authorization: campaigngateway.NewAuthorizationGateway(deps.Participants.Authorization),
		},
		Characters: campaigncharacters.ServiceConfig{
			Read: campaignapp.CharacterReadServiceConfig{
				Read:               campaigngateway.NewCharacterReadGateway(deps.Characters.Read, assetBaseURL),
				BatchAuthorization: campaigngateway.NewBatchAuthorizationGateway(deps.Characters.Authorization),
			},
			Ownership: campaignapp.CharacterOwnershipServiceConfig{
				Read:         campaigngateway.NewCharacterReadGateway(deps.Characters.Read, assetBaseURL),
				Mutation:     campaigngateway.NewCharacterOwnershipMutationGateway(deps.Characters.Ownership),
				Participants: campaigngateway.NewParticipantReadGateway(deps.Characters.Participants, assetBaseURL),
				Sessions:     campaigngateway.NewSessionReadGateway(deps.Characters.Sessions),
			},
			Mutation: campaignapp.CharacterMutationServiceConfig{
				Mutation: campaigngateway.NewCharacterMutationGateway(deps.Characters.Mutation),
				Sessions: campaigngateway.NewSessionReadGateway(deps.Characters.Sessions),
			},
			Creation: campaignapp.CharacterCreationServiceConfig{
				Read:     campaigngateway.NewCharacterCreationReadGateway(deps.Characters.CreationRead, assetBaseURL),
				Mutation: campaigngateway.NewCharacterCreationMutationGateway(deps.Characters.CreationMutation),
			},
			Authorization: campaigngateway.NewAuthorizationGateway(deps.Characters.Authorization),
		},
		Sessions: campaignsessions.ServiceConfig{
			Characters: campaignapp.CharacterReadServiceConfig{
				Read:               campaigngateway.NewCharacterReadGateway(deps.Characters.Read, assetBaseURL),
				BatchAuthorization: campaigngateway.NewBatchAuthorizationGateway(deps.Characters.Authorization),
			},
			Mutation: campaignapp.SessionMutationServiceConfig{
				Mutation: campaigngateway.NewSessionMutationGateway(deps.Sessions.Mutation),
			},
			Participants:  participantRead,
			Authorization: authorization,
		},
		Invites: campaigninvites.ServiceConfig{
			Read: campaignapp.InviteReadServiceConfig{
				Read: campaigngateway.NewInviteReadGateway(deps.Invites.Read),
			},
			Mutation: campaignapp.InviteMutationServiceConfig{
				Mutation: campaigngateway.NewInviteMutationGateway(deps.Invites.Mutation),
			},
			ParticipantRead: participantRead,
			Authorization:   campaigngateway.NewAuthorizationGateway(deps.Invites.Authorization),
		},
	}
}

func configWithGRPCDeps(deps campaigngateway.GRPCGatewayDeps, base modulehandler.Base, workflows campaignworkflow.Registry) Config {
	return Config{
		Services:         newHandlerServices(serviceConfigsWithGRPCDeps(deps, ""), workflows),
		Base:             base,
		PlayFallbackPort: "",
		PlayLaunchGrant:  fakePlayLaunchGrantConfig(),
		Systems:          newTestCampaignSystems(workflows),
	}
}

func configWithGatewayAndSync(
	gateway testGatewayBundle,
	base modulehandler.Base,
	workflows campaignworkflow.Registry,
	sync campaigndetail.DashboardSync,
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
