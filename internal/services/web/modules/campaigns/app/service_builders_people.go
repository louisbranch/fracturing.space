package app

// participantReadService owns participant reads, edits, and row-level
// authorization hydration.
type participantReadService struct {
	read               CampaignParticipantReadGateway
	workspace          CampaignWorkspaceReadGateway
	batchAuthorization BatchAuthorizationGateway
	auth               authorizationSupport
}

// participantMutationService owns participant create/update mutations.
type participantMutationService struct {
	read      CampaignParticipantReadGateway
	mutation  CampaignParticipantMutationGateway
	workspace CampaignWorkspaceReadGateway
	auth      authorizationSupport
}

// automationReadService keeps campaign automation editor reads separate from mutations.
type automationReadService struct {
	participants CampaignParticipantReadGateway
	read         CampaignAutomationReadGateway
	auth         authorizationSupport
}

// automationMutationService keeps campaign automation mutations separate from editor reads.
type automationMutationService struct {
	participants CampaignParticipantReadGateway
	mutation     CampaignAutomationMutationGateway
	auth         authorizationSupport
}

// characterReadService owns character entity/list/editor reads.
type characterReadService struct {
	read               CampaignCharacterReadGateway
	batchAuthorization BatchAuthorizationGateway
	auth               authorizationSupport
}

// characterControlService owns character-control detail state and control mutations.
type characterControlService struct {
	read         CampaignCharacterReadGateway
	mutation     CampaignCharacterControlMutationGateway
	participants CampaignParticipantReadGateway
	sessions     CampaignSessionReadGateway
	auth         authorizationSupport
}

// characterMutationService owns character create/update/delete mutations.
type characterMutationService struct {
	mutation CampaignCharacterMutationGateway
	sessions CampaignSessionReadGateway
	auth     authorizationSupport
}

// NewParticipantReadService constructs the participant read service surface
// from explicit gateway seams.
func NewParticipantReadService(config ParticipantReadServiceConfig, authorization AuthorizationGateway) CampaignParticipantReadService {
	if config.Read == nil || config.Workspace == nil || config.BatchAuthorization == nil || authorization == nil {
		return nil
	}
	return participantReadService{
		read:               config.Read,
		workspace:          config.Workspace,
		batchAuthorization: config.BatchAuthorization,
		auth:               authorizationSupport{gateway: authorization},
	}
}

// NewParticipantMutationService constructs the participant mutation service
// surface from explicit gateway seams.
func NewParticipantMutationService(config ParticipantMutationServiceConfig, authorization AuthorizationGateway) CampaignParticipantMutationService {
	if config.Read == nil || config.Mutation == nil || config.Workspace == nil || authorization == nil {
		return nil
	}
	return participantMutationService{
		read:      config.Read,
		mutation:  config.Mutation,
		workspace: config.Workspace,
		auth:      authorizationSupport{gateway: authorization},
	}
}

// NewAutomationReadService constructs the campaign automation read surface from
// explicit gateway seams.
func NewAutomationReadService(config AutomationReadServiceConfig, authorization AuthorizationGateway) CampaignAutomationReadService {
	if config.Participants == nil || config.Read == nil || authorization == nil {
		return nil
	}
	return automationReadService{
		participants: config.Participants,
		read:         config.Read,
		auth:         authorizationSupport{gateway: authorization},
	}
}

// NewAutomationMutationService constructs the campaign automation mutation
// surface from explicit gateway seams.
func NewAutomationMutationService(config AutomationMutationServiceConfig, authorization AuthorizationGateway) CampaignAutomationMutationService {
	if config.Participants == nil || config.Mutation == nil || authorization == nil {
		return nil
	}
	return automationMutationService{
		participants: config.Participants,
		mutation:     config.Mutation,
		auth:         authorizationSupport{gateway: authorization},
	}
}

// NewCharacterReadService constructs the character read service surface from
// explicit gateway seams.
func NewCharacterReadService(config CharacterReadServiceConfig, authorization AuthorizationGateway) CampaignCharacterReadService {
	if config.Read == nil || config.BatchAuthorization == nil || authorization == nil {
		return nil
	}
	return characterReadService{
		read:               config.Read,
		batchAuthorization: config.BatchAuthorization,
		auth:               authorizationSupport{gateway: authorization},
	}
}

// NewCharacterControlService constructs the character-control service surface
// from explicit gateway seams.
func NewCharacterControlService(config CharacterControlServiceConfig, authorization AuthorizationGateway) CampaignCharacterControlService {
	if config.Read == nil || config.Mutation == nil || config.Participants == nil || config.Sessions == nil || authorization == nil {
		return nil
	}
	return characterControlService{
		read:         config.Read,
		mutation:     config.Mutation,
		participants: config.Participants,
		sessions:     config.Sessions,
		auth:         authorizationSupport{gateway: authorization},
	}
}

// NewCharacterMutationService constructs the character mutation service
// surface from explicit gateway seams.
func NewCharacterMutationService(config CharacterMutationServiceConfig, authorization AuthorizationGateway) CampaignCharacterMutationService {
	if config.Mutation == nil || config.Sessions == nil || authorization == nil {
		return nil
	}
	return characterMutationService{
		mutation: config.Mutation,
		sessions: config.Sessions,
		auth:     authorizationSupport{gateway: authorization},
	}
}
