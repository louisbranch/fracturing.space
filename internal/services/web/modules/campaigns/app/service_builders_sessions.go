package app

import "errors"

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

// NewSessionReadService constructs the session read service surface from
// explicit gateway seams. Returns an error when the read gateway is absent.
func NewSessionReadService(config SessionReadServiceConfig) (CampaignSessionReadService, error) {
	if config.Read == nil {
		return nil, errors.New("session read service: missing required read gateway")
	}
	return sessionReadService{
		read: config.Read,
	}, nil
}

// NewSessionMutationService constructs the session mutation service surface
// from explicit gateway seams. Returns an error when required dependencies
// are absent.
func NewSessionMutationService(config SessionMutationServiceConfig, authorization AuthorizationGateway) (CampaignSessionMutationService, error) {
	if config.Mutation == nil || authorization == nil {
		return nil, errors.New("session mutation service: missing required dependencies")
	}
	return sessionMutationService{
		mutation: config.Mutation,
		auth:     authorizationSupport{gateway: authorization},
	}, nil
}

// NewInviteReadService constructs the invite read service surface from
// explicit gateway seams. Returns an error when required dependencies are
// absent.
func NewInviteReadService(config InviteReadServiceConfig, authorization AuthorizationGateway) (CampaignInviteReadService, error) {
	if config.Read == nil || authorization == nil {
		return nil, errors.New("invite read service: missing required dependencies")
	}
	return inviteReadService{
		read: config.Read,
		auth: authorizationSupport{gateway: authorization},
	}, nil
}

// NewInviteMutationService constructs the invite mutation service surface from
// explicit gateway seams. Returns an error when required dependencies are
// absent.
func NewInviteMutationService(config InviteMutationServiceConfig, authorization AuthorizationGateway) (CampaignInviteMutationService, error) {
	if config.Mutation == nil || authorization == nil {
		return nil, errors.New("invite mutation service: missing required dependencies")
	}
	return inviteMutationService{
		mutation: config.Mutation,
		auth:     authorizationSupport{gateway: authorization},
	}, nil
}

// NewConfigurationService constructs the campaign-configuration service
// surface from explicit gateway seams. Returns an error when required
// dependencies are absent.
func NewConfigurationService(config ConfigurationServiceConfig, authorization AuthorizationGateway) (CampaignConfigurationService, error) {
	if config.Workspace == nil || config.Mutation == nil || authorization == nil {
		return nil, errors.New("configuration service: missing required dependencies")
	}
	return configurationService{
		workspace: config.Workspace,
		mutation:  config.Mutation,
		auth:      authorizationSupport{gateway: authorization},
	}, nil
}
