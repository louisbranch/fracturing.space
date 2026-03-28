package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigncharacters "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/characters"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
	campaignoverview "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/overview"
	campaignparticipants "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/participants"
)

// newOverviewSurfaceConfig keeps overview/configuration/automation composition
// local to the overview route surface.
func newOverviewSurfaceConfig(config CompositionConfig) campaignoverview.ServiceConfig {
	return campaignoverview.ServiceConfig{
		AutomationRead:     newAutomationReadServiceConfig(config),
		AutomationMutation: newAutomationMutationServiceConfig(config),
		Configuration:      newConfigurationServiceConfig(config),
		Authorization:      campaigngateway.NewAuthorizationGateway(config.Gateway.Overview.Authorization),
	}
}

// newParticipantSurfaceConfig keeps participant composition local to the
// participant route surface.
func newParticipantSurfaceConfig(config CompositionConfig) campaignparticipants.ServiceConfig {
	return campaignparticipants.ServiceConfig{
		Read:          newParticipantReadServiceConfig(config),
		Mutation:      newParticipantMutationServiceConfig(config),
		Authorization: campaigngateway.NewAuthorizationGateway(config.Gateway.Participants.Authorization),
	}
}

// newCharacterSurfaceConfig keeps character/ownership/creation composition local
// to the character route surface.
func newCharacterSurfaceConfig(config CompositionConfig) campaigncharacters.ServiceConfig {
	return campaigncharacters.ServiceConfig{
		Read:          newCharacterReadServiceConfig(config),
		Ownership:     newCharacterOwnershipServiceConfig(config),
		Mutation:      newCharacterMutationServiceConfig(config),
		Creation:      newCharacterCreationServiceConfig(config),
		Authorization: campaigngateway.NewAuthorizationGateway(config.Gateway.Characters.Authorization),
	}
}

// newParticipantReadServiceConfig keeps participant read/editor wiring local
// to the participant capability family.
func newParticipantReadServiceConfig(config CompositionConfig) campaignapp.ParticipantReadServiceConfig {
	return campaignapp.ParticipantReadServiceConfig{
		Read:               campaigngateway.NewParticipantReadGateway(config.Gateway.Participants.Read, config.Options.AssetBaseURL),
		Workspace:          campaigngateway.NewWorkspaceReadGateway(config.Gateway.Participants.Workspace, config.Options.AssetBaseURL),
		BatchAuthorization: campaigngateway.NewBatchAuthorizationGateway(config.Gateway.Participants.Authorization),
	}
}

// newParticipantMutationServiceConfig keeps participant mutation wiring local
// to the participant capability family.
func newParticipantMutationServiceConfig(config CompositionConfig) campaignapp.ParticipantMutationServiceConfig {
	return campaignapp.ParticipantMutationServiceConfig{
		Read:      campaigngateway.NewParticipantReadGateway(config.Gateway.Participants.Read, config.Options.AssetBaseURL),
		Mutation:  campaigngateway.NewParticipantMutationGateway(config.Gateway.Participants.Mutation),
		Workspace: campaigngateway.NewWorkspaceReadGateway(config.Gateway.Participants.Workspace, config.Options.AssetBaseURL),
	}
}

// newCharacterReadServiceConfig keeps character read composition local to the
// character read capability.
func newCharacterReadServiceConfig(config CompositionConfig) campaignapp.CharacterReadServiceConfig {
	return campaignapp.CharacterReadServiceConfig{
		Read:               campaigngateway.NewCharacterReadGateway(config.Gateway.Characters.Read, config.Options.AssetBaseURL),
		BatchAuthorization: campaigngateway.NewBatchAuthorizationGateway(config.Gateway.Characters.Authorization),
	}
}

// newCharacterOwnershipServiceConfig keeps character-owner composition local
// to the character ownership capability.
func newCharacterOwnershipServiceConfig(config CompositionConfig) campaignapp.CharacterOwnershipServiceConfig {
	return campaignapp.CharacterOwnershipServiceConfig{
		Read:         campaigngateway.NewCharacterReadGateway(config.Gateway.Characters.Read, config.Options.AssetBaseURL),
		Mutation:     campaigngateway.NewCharacterOwnershipMutationGateway(config.Gateway.Characters.Ownership),
		Participants: campaigngateway.NewParticipantReadGateway(config.Gateway.Characters.Participants, config.Options.AssetBaseURL),
		Sessions:     campaigngateway.NewSessionReadGateway(config.Gateway.Characters.Sessions),
	}
}

// newCharacterMutationServiceConfig keeps create/update/delete composition
// local to the character mutation capability.
func newCharacterMutationServiceConfig(config CompositionConfig) campaignapp.CharacterMutationServiceConfig {
	return campaignapp.CharacterMutationServiceConfig{
		Mutation: campaigngateway.NewCharacterMutationGateway(config.Gateway.Characters.Mutation),
		Sessions: campaigngateway.NewSessionReadGateway(config.Gateway.Characters.Sessions),
	}
}

// newAutomationReadServiceConfig keeps participant-adjacent automation editor
// wiring explicit without widening participant composition.
func newAutomationReadServiceConfig(config CompositionConfig) campaignapp.AutomationReadServiceConfig {
	return campaignapp.AutomationReadServiceConfig{
		Participants: campaigngateway.NewParticipantReadGateway(config.Gateway.Overview.Participants, config.Options.AssetBaseURL),
		Read:         campaigngateway.NewAutomationReadGateway(config.Gateway.Overview.AutomationRead),
	}
}

// newAutomationMutationServiceConfig keeps participant-adjacent automation
// mutation wiring explicit without widening configuration composition.
func newAutomationMutationServiceConfig(config CompositionConfig) campaignapp.AutomationMutationServiceConfig {
	return campaignapp.AutomationMutationServiceConfig{
		Participants: campaigngateway.NewParticipantReadGateway(config.Gateway.Overview.Participants, config.Options.AssetBaseURL),
		Mutation:     campaigngateway.NewAutomationMutationGateway(config.Gateway.Overview.AutomationMutation),
	}
}

// newOverviewGatewayDeps groups overview/configuration/automation clients by
// the overview surface that owns them.
func newOverviewGatewayDeps(deps Dependencies) campaigngateway.OverviewGatewayDeps {
	return campaigngateway.OverviewGatewayDeps{
		Participants:          campaigngateway.ParticipantReadDeps{Participant: deps.ParticipantClient},
		Workspace:             campaigngateway.WorkspaceReadDeps{Campaign: deps.CampaignClient},
		Authorization:         campaigngateway.AuthorizationDeps{Client: deps.AuthorizationClient},
		AutomationRead:        campaigngateway.AutomationReadDeps{Agent: deps.AgentClient},
		AutomationMutation:    campaigngateway.AutomationMutationDeps{Campaign: deps.CampaignClient},
		ConfigurationMutation: campaigngateway.ConfigurationMutationDeps{Campaign: deps.CampaignClient},
	}
}

// newParticipantGatewayDeps groups participant editor clients by the
// participant surface that consumes them.
func newParticipantGatewayDeps(deps Dependencies) campaigngateway.ParticipantGatewayDeps {
	return campaigngateway.ParticipantGatewayDeps{
		Read:          campaigngateway.ParticipantReadDeps{Participant: deps.ParticipantClient},
		Mutation:      campaigngateway.ParticipantMutationDeps{Participant: deps.ParticipantClient},
		Workspace:     campaigngateway.WorkspaceReadDeps{Campaign: deps.CampaignClient},
		Authorization: campaigngateway.AuthorizationDeps{Client: deps.AuthorizationClient},
	}
}

// newCharacterGatewayDeps groups character read/ownership/mutation clients by
// the character surface that consumes them.
func newCharacterGatewayDeps(deps Dependencies) campaigngateway.CharacterGatewayDeps {
	return campaigngateway.CharacterGatewayDeps{
		Read: campaigngateway.CharacterReadDeps{
			Character:          deps.CharacterClient,
			Participant:        deps.ParticipantClient,
			DaggerheartContent: deps.DaggerheartContentClient,
		},
		Ownership:        campaigngateway.CharacterOwnershipMutationDeps{Character: deps.CharacterClient},
		Mutation:         campaigngateway.CharacterMutationDeps{Character: deps.CharacterClient},
		Participants:     campaigngateway.ParticipantReadDeps{Participant: deps.ParticipantClient},
		Sessions:         campaigngateway.SessionReadDeps{Session: deps.SessionClient, Campaign: deps.CampaignClient},
		Authorization:    campaigngateway.AuthorizationDeps{Client: deps.AuthorizationClient},
		CreationRead:     campaigngateway.CharacterCreationReadDeps{Character: deps.CharacterClient, DaggerheartContent: deps.DaggerheartContentClient, DaggerheartAsset: deps.DaggerheartAssetClient},
		CreationMutation: campaigngateway.CharacterCreationMutationDeps{Character: deps.CharacterClient},
	}
}
