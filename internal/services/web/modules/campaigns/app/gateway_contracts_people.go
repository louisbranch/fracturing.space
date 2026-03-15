package app

import "context"

// CampaignParticipantReadGateway loads participant reads for the web service.
type CampaignParticipantReadGateway interface {
	CampaignParticipants(context.Context, string) ([]CampaignParticipant, error)
	CampaignParticipant(context.Context, string, string) (CampaignParticipant, error)
}

// CampaignCharacterReadGateway loads character reads for the web service.
type CampaignCharacterReadGateway interface {
	CampaignCharacters(context.Context, string, CharacterReadContext) ([]CampaignCharacter, error)
	CampaignCharacter(context.Context, string, string, CharacterReadContext) (CampaignCharacter, error)
}

// CampaignAutomationReadGateway loads campaign automation reads for the web service.
type CampaignAutomationReadGateway interface {
	CampaignAIAgents(context.Context) ([]CampaignAIAgentOption, error)
}

// CampaignAutomationMutationGateway applies campaign-level automation mutations for the web service.
type CampaignAutomationMutationGateway interface {
	UpdateCampaignAIBinding(context.Context, string, UpdateCampaignAIBindingInput) error
}

// CampaignCharacterControlMutationGateway applies character-controller mutations for the web service.
type CampaignCharacterControlMutationGateway interface {
	SetCharacterController(context.Context, string, string, string) error
	ClaimCharacterControl(context.Context, string, string) error
	ReleaseCharacterControl(context.Context, string, string) error
}

// CampaignCharacterMutationGateway applies character create/update/delete mutations for the web service.
type CampaignCharacterMutationGateway interface {
	CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error)
	UpdateCharacter(context.Context, string, string, UpdateCharacterInput) error
	DeleteCharacter(context.Context, string, string) error
}

// CampaignParticipantMutationGateway applies participant mutations for the web service.
type CampaignParticipantMutationGateway interface {
	CreateParticipant(context.Context, string, CreateParticipantInput) (CreateParticipantResult, error)
	UpdateParticipant(context.Context, string, UpdateParticipantInput) error
}
