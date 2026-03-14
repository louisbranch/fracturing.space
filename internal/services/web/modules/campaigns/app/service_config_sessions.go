package app

// SessionReadServiceConfig keeps session read dependencies explicit.
type SessionReadServiceConfig struct {
	Read CampaignSessionReadGateway
}

// SessionMutationServiceConfig keeps session mutation dependencies explicit.
type SessionMutationServiceConfig struct {
	Mutation CampaignSessionMutationGateway
}

// InviteReadServiceConfig keeps invite read/search dependencies explicit.
type InviteReadServiceConfig struct {
	Read CampaignInviteReadGateway
}

// InviteMutationServiceConfig keeps invite mutation dependencies explicit.
type InviteMutationServiceConfig struct {
	Mutation CampaignInviteMutationGateway
}

// ConfigurationServiceConfig keeps campaign-configuration dependencies explicit.
type ConfigurationServiceConfig struct {
	Workspace CampaignWorkspaceReadGateway
	Mutation  CampaignConfigurationMutationGateway
}
