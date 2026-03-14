package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
)

// newParticipantReadServiceConfig keeps participant read/editor wiring local
// to the participant capability family.
func newParticipantReadServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.ParticipantReadServiceConfig {
	return campaignapp.ParticipantReadServiceConfig{
		Read:               campaigngateway.NewParticipantReadGateway(deps.ParticipantRead, assetBaseURL),
		Workspace:          campaigngateway.NewWorkspaceReadGateway(deps.WorkspaceRead, assetBaseURL),
		BatchAuthorization: campaigngateway.NewBatchAuthorizationGateway(deps.Authorization),
	}
}

// newParticipantMutationServiceConfig keeps participant mutation wiring local
// to the participant capability family.
func newParticipantMutationServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.ParticipantMutationServiceConfig {
	return campaignapp.ParticipantMutationServiceConfig{
		Read:      campaigngateway.NewParticipantReadGateway(deps.ParticipantRead, assetBaseURL),
		Mutation:  campaigngateway.NewParticipantMutationGateway(deps.ParticipantMutate),
		Workspace: campaigngateway.NewWorkspaceReadGateway(deps.WorkspaceRead, assetBaseURL),
	}
}

// newCharacterReadServiceConfig keeps character read composition local to the
// character read capability.
func newCharacterReadServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.CharacterReadServiceConfig {
	return campaignapp.CharacterReadServiceConfig{
		Read:               campaigngateway.NewCharacterReadGateway(deps.CharacterRead, assetBaseURL),
		BatchAuthorization: campaigngateway.NewBatchAuthorizationGateway(deps.Authorization),
	}
}

// newCharacterControlServiceConfig keeps character-control composition local
// to the character control capability.
func newCharacterControlServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.CharacterControlServiceConfig {
	return campaignapp.CharacterControlServiceConfig{
		Read:         campaigngateway.NewCharacterReadGateway(deps.CharacterRead, assetBaseURL),
		Mutation:     campaigngateway.NewCharacterControlMutationGateway(deps.CharacterControl),
		Participants: campaigngateway.NewParticipantReadGateway(deps.ParticipantRead, assetBaseURL),
		Sessions:     campaigngateway.NewSessionReadGateway(deps.SessionRead),
	}
}

// newCharacterMutationServiceConfig keeps create/update/delete composition
// local to the character mutation capability.
func newCharacterMutationServiceConfig(deps campaigngateway.GRPCGatewayDeps) campaignapp.CharacterMutationServiceConfig {
	return campaignapp.CharacterMutationServiceConfig{
		Mutation: campaigngateway.NewCharacterMutationGateway(deps.CharacterMutate),
		Sessions: campaigngateway.NewSessionReadGateway(deps.SessionRead),
	}
}

// newAutomationReadServiceConfig keeps participant-adjacent automation editor
// wiring explicit without widening participant composition.
func newAutomationReadServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.AutomationReadServiceConfig {
	return campaignapp.AutomationReadServiceConfig{
		Participants: campaigngateway.NewParticipantReadGateway(deps.ParticipantRead, assetBaseURL),
		Read:         campaigngateway.NewAutomationReadGateway(deps.AutomationRead),
	}
}

// newAutomationMutationServiceConfig keeps participant-adjacent automation
// mutation wiring explicit without widening configuration composition.
func newAutomationMutationServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.AutomationMutationServiceConfig {
	return campaignapp.AutomationMutationServiceConfig{
		Participants: campaigngateway.NewParticipantReadGateway(deps.ParticipantRead, assetBaseURL),
		Mutation:     campaigngateway.NewAutomationMutationGateway(deps.AutomationMutate),
	}
}
