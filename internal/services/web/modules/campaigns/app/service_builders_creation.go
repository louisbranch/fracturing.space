package app

import "errors"

// creationPageService owns read-only character-creation workflow inputs.
type creationPageService struct {
	read CharacterCreationReadGateway
}

// creationMutationService owns character-creation step mutations and reset flow.
type creationMutationService struct {
	read     CharacterCreationReadGateway
	mutation CharacterCreationMutationGateway
	auth     authorizationSupport
}

// NewCharacterCreationPageService constructs the character-creation page
// service surface from explicit gateway seams. Returns an error when the read
// gateway is absent.
func NewCharacterCreationPageService(config CharacterCreationServiceConfig) (CampaignCharacterCreationPageService, error) {
	if config.Read == nil {
		return nil, errors.New("character creation page service: missing required read gateway")
	}
	return creationPageService{read: config.Read}, nil
}

// NewCharacterCreationMutationService constructs the character-creation
// mutation service surface from explicit gateway seams. Returns an error when
// required dependencies are absent.
func NewCharacterCreationMutationService(config CharacterCreationServiceConfig, authorization AuthorizationGateway) (CampaignCharacterCreationMutationService, error) {
	if config.Read == nil || config.Mutation == nil || authorization == nil {
		return nil, errors.New("character creation mutation service: missing required dependencies")
	}
	return creationMutationService{
		read:     config.Read,
		mutation: config.Mutation,
		auth:     authorizationSupport{gateway: authorization},
	}, nil
}
