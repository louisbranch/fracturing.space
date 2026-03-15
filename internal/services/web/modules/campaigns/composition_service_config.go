package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
)

// newServiceConfigFromGRPCDeps builds the campaigns app config from explicit
// generated-client dependency groups.
func newServiceConfigFromGRPCDeps(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.ServiceConfig {
	return campaignapp.ServiceConfig{
		Catalog:             newCatalogServiceConfig(deps, assetBaseURL),
		Starter:             newStarterServiceConfig(deps),
		Workspace:           newWorkspaceServiceConfig(deps, assetBaseURL),
		Game:                newGameServiceConfig(deps),
		ParticipantRead:     newParticipantReadServiceConfig(deps, assetBaseURL),
		ParticipantMutation: newParticipantMutationServiceConfig(deps, assetBaseURL),
		CharacterRead:       newCharacterReadServiceConfig(deps, assetBaseURL),
		CharacterControl:    newCharacterControlServiceConfig(deps, assetBaseURL),
		CharacterMutation:   newCharacterMutationServiceConfig(deps),
		SessionRead:         newSessionReadServiceConfig(deps),
		SessionMutation:     newSessionMutationServiceConfig(deps),
		InviteRead:          newInviteReadServiceConfig(deps),
		InviteMutation:      newInviteMutationServiceConfig(deps),
		Configuration:       newConfigurationServiceConfig(deps, assetBaseURL),
		AutomationRead:      newAutomationReadServiceConfig(deps, assetBaseURL),
		AutomationMutation:  newAutomationMutationServiceConfig(deps, assetBaseURL),
		Creation:            newCharacterCreationServiceConfig(deps, assetBaseURL),
		Authorization:       campaigngateway.NewAuthorizationGateway(deps.Authorization),
	}
}
