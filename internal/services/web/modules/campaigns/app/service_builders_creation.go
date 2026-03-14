package app

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
// service surface from explicit gateway seams.
func NewCharacterCreationPageService(config CharacterCreationServiceConfig) CampaignCharacterCreationPageService {
	if config.Read == nil {
		return nil
	}
	return creationPageService{read: config.Read}
}

// NewCharacterCreationMutationService constructs the character-creation
// mutation service surface from explicit gateway seams.
func NewCharacterCreationMutationService(config CharacterCreationServiceConfig, authorization AuthorizationGateway) CampaignCharacterCreationMutationService {
	if config.Read == nil || config.Mutation == nil || authorization == nil {
		return nil
	}
	return creationMutationService{
		read:     config.Read,
		mutation: config.Mutation,
		auth:     authorizationSupport{gateway: authorization},
	}
}
