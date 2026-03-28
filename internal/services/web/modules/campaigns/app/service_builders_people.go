package app

import "errors"

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

// characterOwnershipService owns character-owner detail state and ownership mutations.
type characterOwnershipService struct {
	read         CampaignCharacterReadGateway
	mutation     CampaignCharacterOwnershipMutationGateway
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
// from explicit gateway seams. Returns an error when required dependencies
// are absent.
func NewParticipantReadService(config ParticipantReadServiceConfig, authorization AuthorizationGateway) (CampaignParticipantReadService, error) {
	if config.Read == nil || config.Workspace == nil || config.BatchAuthorization == nil || authorization == nil {
		return nil, errors.New("participant read service: missing required dependencies")
	}
	return participantReadService{
		read:               config.Read,
		workspace:          config.Workspace,
		batchAuthorization: config.BatchAuthorization,
		auth:               authorizationSupport{gateway: authorization},
	}, nil
}

// NewParticipantMutationService constructs the participant mutation service
// surface from explicit gateway seams. Returns an error when required
// dependencies are absent.
func NewParticipantMutationService(config ParticipantMutationServiceConfig, authorization AuthorizationGateway) (CampaignParticipantMutationService, error) {
	if config.Read == nil || config.Mutation == nil || config.Workspace == nil || authorization == nil {
		return nil, errors.New("participant mutation service: missing required dependencies")
	}
	return participantMutationService{
		read:      config.Read,
		mutation:  config.Mutation,
		workspace: config.Workspace,
		auth:      authorizationSupport{gateway: authorization},
	}, nil
}

// NewAutomationReadService constructs the campaign automation read surface from
// explicit gateway seams. Returns an error when required dependencies are
// absent.
func NewAutomationReadService(config AutomationReadServiceConfig, authorization AuthorizationGateway) (CampaignAutomationReadService, error) {
	if config.Participants == nil || config.Read == nil || authorization == nil {
		return nil, errors.New("automation read service: missing required dependencies")
	}
	return automationReadService{
		participants: config.Participants,
		read:         config.Read,
		auth:         authorizationSupport{gateway: authorization},
	}, nil
}

// NewAutomationMutationService constructs the campaign automation mutation
// surface from explicit gateway seams. Returns an error when required
// dependencies are absent.
func NewAutomationMutationService(config AutomationMutationServiceConfig, authorization AuthorizationGateway) (CampaignAutomationMutationService, error) {
	if config.Participants == nil || config.Mutation == nil || authorization == nil {
		return nil, errors.New("automation mutation service: missing required dependencies")
	}
	return automationMutationService{
		participants: config.Participants,
		mutation:     config.Mutation,
		auth:         authorizationSupport{gateway: authorization},
	}, nil
}

// NewCharacterReadService constructs the character read service surface from
// explicit gateway seams. Returns an error when required dependencies are
// absent.
func NewCharacterReadService(config CharacterReadServiceConfig, authorization AuthorizationGateway) (CampaignCharacterReadService, error) {
	if config.Read == nil || config.BatchAuthorization == nil || authorization == nil {
		return nil, errors.New("character read service: missing required dependencies")
	}
	return characterReadService{
		read:               config.Read,
		batchAuthorization: config.BatchAuthorization,
		auth:               authorizationSupport{gateway: authorization},
	}, nil
}

// NewCharacterOwnershipService constructs the character-owner service surface
// from explicit gateway seams. Returns an error when required dependencies are
// absent.
func NewCharacterOwnershipService(config CharacterOwnershipServiceConfig, authorization AuthorizationGateway) (CampaignCharacterOwnershipService, error) {
	if config.Read == nil || config.Mutation == nil || config.Participants == nil || config.Sessions == nil || authorization == nil {
		return nil, errors.New("character ownership service: missing required dependencies")
	}
	return characterOwnershipService{
		read:         config.Read,
		mutation:     config.Mutation,
		participants: config.Participants,
		sessions:     config.Sessions,
		auth:         authorizationSupport{gateway: authorization},
	}, nil
}

// NewCharacterMutationService constructs the character mutation service
// surface from explicit gateway seams. Returns an error when required
// dependencies are absent.
func NewCharacterMutationService(config CharacterMutationServiceConfig, authorization AuthorizationGateway) (CampaignCharacterMutationService, error) {
	if config.Mutation == nil || config.Sessions == nil || authorization == nil {
		return nil, errors.New("character mutation service: missing required dependencies")
	}
	return characterMutationService{
		mutation: config.Mutation,
		sessions: config.Sessions,
		auth:     authorizationSupport{gateway: authorization},
	}, nil
}
