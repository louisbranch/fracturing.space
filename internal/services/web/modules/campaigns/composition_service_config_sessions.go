package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
)

// newSessionReadServiceConfig keeps session read composition with the session
// capability family.
func newSessionReadServiceConfig(deps campaigngateway.GRPCGatewayDeps) campaignapp.SessionReadServiceConfig {
	return campaignapp.SessionReadServiceConfig{
		Read: campaigngateway.NewSessionReadGateway(deps.SessionRead),
	}
}

// newSessionMutationServiceConfig keeps session mutation composition with the
// session capability family.
func newSessionMutationServiceConfig(deps campaigngateway.GRPCGatewayDeps) campaignapp.SessionMutationServiceConfig {
	return campaignapp.SessionMutationServiceConfig{
		Mutation: campaigngateway.NewSessionMutationGateway(deps.SessionMutate),
	}
}

// newInviteReadServiceConfig keeps invite read/search wiring local to the
// invite capability family.
func newInviteReadServiceConfig(deps campaigngateway.GRPCGatewayDeps) campaignapp.InviteReadServiceConfig {
	return campaignapp.InviteReadServiceConfig{
		Read: campaigngateway.NewInviteReadGateway(deps.InviteRead),
	}
}

// newInviteMutationServiceConfig keeps invite mutation wiring local to the
// invite capability family.
func newInviteMutationServiceConfig(deps campaigngateway.GRPCGatewayDeps) campaignapp.InviteMutationServiceConfig {
	return campaignapp.InviteMutationServiceConfig{
		Mutation: campaigngateway.NewInviteMutationGateway(deps.InviteMutate),
	}
}

// newConfigurationServiceConfig keeps campaign settings composition distinct
// from automation and participant wiring.
func newConfigurationServiceConfig(deps campaigngateway.GRPCGatewayDeps, assetBaseURL string) campaignapp.ConfigurationServiceConfig {
	return campaignapp.ConfigurationServiceConfig{
		Workspace: campaigngateway.NewWorkspaceReadGateway(deps.WorkspaceRead, assetBaseURL),
		Mutation:  campaigngateway.NewConfigurationMutationGateway(deps.ConfigMutate),
	}
}
