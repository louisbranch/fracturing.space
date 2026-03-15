package campaigns

import (
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
)

// newServiceConfigsFromGRPCDeps builds campaigns app config grouped by owned
// route surface from explicit generated-client dependency groups.
func newServiceConfigsFromGRPCDeps(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) serviceConfigs {
	authorization := campaigngateway.NewAuthorizationGateway(deps.Authorization)
	participantRead := newParticipantReadServiceConfig(deps, assetBaseURL)
	return serviceConfigs{
		Page: pageServiceConfig{
			Workspace:     newWorkspaceServiceConfig(deps, assetBaseURL),
			SessionRead:   newSessionReadServiceConfig(deps),
			Authorization: authorization,
		},
		Catalog: catalogServiceConfig{
			Catalog: newCatalogServiceConfig(deps, assetBaseURL),
		},
		Starter: starterServiceConfig{
			Starter: newStarterServiceConfig(deps),
		},
		Overview: overviewServiceConfig{
			AutomationRead:     newAutomationReadServiceConfig(deps, assetBaseURL),
			AutomationMutation: newAutomationMutationServiceConfig(deps, assetBaseURL),
			Configuration:      newConfigurationServiceConfig(deps, assetBaseURL),
			Authorization:      authorization,
		},
		Participants: participantServiceConfig{
			Read:          participantRead,
			Mutation:      newParticipantMutationServiceConfig(deps, assetBaseURL),
			Authorization: authorization,
		},
		Characters: characterServiceConfig{
			Read:          newCharacterReadServiceConfig(deps, assetBaseURL),
			Control:       newCharacterControlServiceConfig(deps, assetBaseURL),
			Mutation:      newCharacterMutationServiceConfig(deps),
			Creation:      newCharacterCreationServiceConfig(deps, assetBaseURL),
			Authorization: authorization,
		},
		Sessions: sessionServiceConfig{
			Mutation: newSessionMutationServiceConfig(deps),
		},
		Invites: inviteServiceConfig{
			Read:            newInviteReadServiceConfig(deps),
			Mutation:        newInviteMutationServiceConfig(deps),
			ParticipantRead: participantRead,
			Authorization:   authorization,
		},
	}
}
