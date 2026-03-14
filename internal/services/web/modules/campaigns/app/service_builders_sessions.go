package app

// sessionReadService keeps session reads on one capability seam.
type sessionReadService struct {
	read CampaignSessionReadGateway
}

// sessionMutationService keeps session lifecycle mutations on one capability seam.
type sessionMutationService struct {
	mutation CampaignSessionMutationGateway
	auth     authorizationSupport
}

// inviteReadService owns invite reads and search.
type inviteReadService struct {
	read CampaignInviteReadGateway
	auth authorizationSupport
}

// inviteMutationService owns invite mutations.
type inviteMutationService struct {
	mutation CampaignInviteMutationGateway
	auth     authorizationSupport
}

// configurationService owns campaign settings updates without widening into automation or participant editing.
type configurationService struct {
	workspace CampaignWorkspaceReadGateway
	mutation  CampaignConfigurationMutationGateway
	auth      authorizationSupport
}

// NewSessionReadService constructs the session read service surface from explicit
// gateway seams.
func NewSessionReadService(config SessionReadServiceConfig) CampaignSessionReadService {
	if config.Read == nil {
		return nil
	}
	return sessionReadService{
		read: config.Read,
	}
}

// NewSessionMutationService constructs the session mutation service surface
// from explicit gateway seams.
func NewSessionMutationService(config SessionMutationServiceConfig, authorization AuthorizationGateway) CampaignSessionMutationService {
	if config.Mutation == nil || authorization == nil {
		return nil
	}
	return sessionMutationService{
		mutation: config.Mutation,
		auth:     authorizationSupport{gateway: authorization},
	}
}

// NewInviteReadService constructs the invite read service surface from explicit gateway
// seams.
func NewInviteReadService(config InviteReadServiceConfig, authorization AuthorizationGateway) CampaignInviteReadService {
	if config.Read == nil || authorization == nil {
		return nil
	}
	return inviteReadService{
		read: config.Read,
		auth: authorizationSupport{gateway: authorization},
	}
}

// NewInviteMutationService constructs the invite mutation service surface from
// explicit gateway seams.
func NewInviteMutationService(config InviteMutationServiceConfig, authorization AuthorizationGateway) CampaignInviteMutationService {
	if config.Mutation == nil || authorization == nil {
		return nil
	}
	return inviteMutationService{
		mutation: config.Mutation,
		auth:     authorizationSupport{gateway: authorization},
	}
}

// NewConfigurationService constructs the campaign-configuration service
// surface from explicit gateway seams.
func NewConfigurationService(config ConfigurationServiceConfig, authorization AuthorizationGateway) CampaignConfigurationService {
	if config.Workspace == nil || config.Mutation == nil || authorization == nil {
		return nil
	}
	return configurationService{
		workspace: config.Workspace,
		mutation:  config.Mutation,
		auth:      authorizationSupport{gateway: authorization},
	}
}
