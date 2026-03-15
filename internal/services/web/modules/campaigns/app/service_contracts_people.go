package app

import "context"

// CampaignParticipantReadService exposes participant-focused reads and editor state.
type CampaignParticipantReadService interface {
	CampaignParticipants(context.Context, string) ([]CampaignParticipant, error)
	CampaignParticipantCreator(context.Context, string) (CampaignParticipantCreator, error)
	CampaignParticipantEditor(context.Context, string, string) (CampaignParticipantEditor, error)
}

// CampaignParticipantMutationService exposes participant create/update mutations.
type CampaignParticipantMutationService interface {
	CreateParticipant(context.Context, string, CreateParticipantInput) (CreateParticipantResult, error)
	UpdateParticipant(context.Context, string, UpdateParticipantInput) error
}

// CampaignAutomationReadService exposes campaign-level AI automation reads.
type CampaignAutomationReadService interface {
	CampaignAIBindingSummary(context.Context, string, string, string) (CampaignAIBindingSummary, error)
	CampaignAIBindingSettings(context.Context, string, string) (CampaignAIBindingSettings, error)
}

// CampaignAutomationMutationService exposes campaign-level AI automation mutations.
type CampaignAutomationMutationService interface {
	UpdateCampaignAIBinding(context.Context, string, UpdateCampaignAIBindingInput) error
}

// CampaignCharacterReadService exposes character list/entity/editor reads.
type CampaignCharacterReadService interface {
	CampaignCharacters(context.Context, string, CharacterReadContext) ([]CampaignCharacter, error)
	CampaignCharacter(context.Context, string, string, CharacterReadContext) (CampaignCharacter, error)
	CampaignCharacterEditor(context.Context, string, string, CharacterReadContext) (CampaignCharacterEditor, error)
}

// CampaignCharacterControlService exposes character-control detail state and control mutations.
type CampaignCharacterControlService interface {
	CampaignCharacterControl(context.Context, string, string, string, CharacterReadContext) (CampaignCharacterControl, error)
	SetCharacterController(context.Context, string, string, string) error
	ClaimCharacterControl(context.Context, string, string, string) error
	ReleaseCharacterControl(context.Context, string, string, string) error
}

// CampaignCharacterMutationService exposes character create/update/delete mutations.
type CampaignCharacterMutationService interface {
	CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error)
	UpdateCharacter(context.Context, string, string, UpdateCharacterInput) error
	DeleteCharacter(context.Context, string, string) error
}
