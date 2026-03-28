package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigngateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/gateway"
)

// newSessionSurfaceConfig keeps session composition local to the session and
// game-launch surface.
func newSessionSurfaceConfig(config CompositionConfig) sessionServiceConfig {
	return sessionServiceConfig{
		Characters:    newCharacterReadServiceConfig(config),
		Mutation:      newSessionMutationServiceConfig(config),
		Participants:  newParticipantReadServiceConfig(config),
		Authorization: newPageAuthorizationGateway(config),
	}
}

// newInviteSurfaceConfig keeps invite composition local to the invite route
// surface.
func newInviteSurfaceConfig(config CompositionConfig) inviteServiceConfig {
	return inviteServiceConfig{
		Read:            newInviteReadServiceConfig(config),
		Mutation:        newInviteMutationServiceConfig(config),
		ParticipantRead: newParticipantReadServiceConfig(config),
		Authorization:   campaigngateway.NewAuthorizationGateway(config.Gateway.Invites.Authorization),
	}
}

// newSessionReadServiceConfig keeps session read composition with the session
// capability family.
func newSessionReadServiceConfig(config CompositionConfig) campaignapp.SessionReadServiceConfig {
	return campaignapp.SessionReadServiceConfig{
		Read: campaigngateway.NewSessionReadGateway(config.Gateway.Page.SessionRead),
	}
}

// newSessionMutationServiceConfig keeps session mutation composition with the
// session capability family.
func newSessionMutationServiceConfig(config CompositionConfig) campaignapp.SessionMutationServiceConfig {
	return campaignapp.SessionMutationServiceConfig{
		Mutation: campaigngateway.NewSessionMutationGateway(config.Gateway.Sessions.Mutation),
	}
}

// newInviteReadServiceConfig keeps invite read/search wiring local to the
// invite capability family.
func newInviteReadServiceConfig(config CompositionConfig) campaignapp.InviteReadServiceConfig {
	return campaignapp.InviteReadServiceConfig{
		Read: campaigngateway.NewInviteReadGateway(config.Gateway.Invites.Read),
	}
}

// newInviteMutationServiceConfig keeps invite mutation wiring local to the
// invite capability family.
func newInviteMutationServiceConfig(config CompositionConfig) campaignapp.InviteMutationServiceConfig {
	return campaignapp.InviteMutationServiceConfig{
		Mutation: campaigngateway.NewInviteMutationGateway(config.Gateway.Invites.Mutation),
	}
}

// newConfigurationServiceConfig keeps campaign settings composition distinct
// from automation and participant wiring.
func newConfigurationServiceConfig(config CompositionConfig) campaignapp.ConfigurationServiceConfig {
	return campaignapp.ConfigurationServiceConfig{
		Workspace: campaigngateway.NewWorkspaceReadGateway(config.Gateway.Overview.Workspace, config.Options.AssetBaseURL),
		Mutation:  campaigngateway.NewConfigurationMutationGateway(config.Gateway.Overview.ConfigurationMutation),
	}
}

// newSessionGatewayDeps groups session mutation clients by the session/game
// launcher surface that owns them.
func newSessionGatewayDeps(deps Dependencies) campaigngateway.SessionGatewayDeps {
	return campaigngateway.SessionGatewayDeps{
		Mutation: campaigngateway.SessionMutationDeps{Session: deps.SessionClient},
	}
}

// newInviteGatewayDeps groups invite read/search/mutation clients by the
// invite surface that owns them.
func newInviteGatewayDeps(deps Dependencies) campaigngateway.InviteGatewayDeps {
	return campaigngateway.InviteGatewayDeps{
		Read: campaigngateway.InviteReadDeps{
			Invite:      deps.InviteClient,
			Participant: deps.ParticipantClient,
			Social:      deps.SocialClient,
			Auth:        deps.AuthClient,
		},
		Mutation:      campaigngateway.InviteMutationDeps{Invite: deps.InviteClient, Auth: deps.AuthClient},
		Participants:  campaigngateway.ParticipantReadDeps{Participant: deps.ParticipantClient},
		Authorization: campaigngateway.AuthorizationDeps{Client: deps.AuthorizationClient},
	}
}
