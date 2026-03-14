package app

import (
	"context"

	"golang.org/x/text/language"
)

// campaignReadGateway defines an internal contract used at this web package boundary.
type campaignReadGateway interface {
	ListCampaigns(context.Context) ([]CampaignSummary, error)
	CampaignName(context.Context, string) (string, error)
	CampaignWorkspace(context.Context, string) (CampaignWorkspace, error)
	CampaignGameSurface(context.Context, string) (CampaignGameSurface, error)
	CampaignAIAgents(context.Context) ([]CampaignAIAgentOption, error)
	CampaignParticipants(context.Context, string) ([]CampaignParticipant, error)
	CampaignParticipant(context.Context, string, string) (CampaignParticipant, error)
	CampaignCharacters(context.Context, string, CampaignCharactersReadOptions) ([]CampaignCharacter, error)
	CampaignSessions(context.Context, string) ([]CampaignSession, error)
	CampaignSessionReadiness(context.Context, string, language.Tag) (CampaignSessionReadiness, error)
	CampaignInvites(context.Context, string) ([]CampaignInvite, error)
	SearchInviteUsers(context.Context, SearchInviteUsersInput) ([]InviteUserSearchResult, error)
	CharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	CharacterCreationCatalog(context.Context, language.Tag) (CampaignCharacterCreationCatalog, error)
	CharacterCreationProfile(context.Context, string, string) (CampaignCharacterCreationProfile, error)
}

// campaignMutationGateway defines an internal contract used at this web package boundary.
type campaignMutationGateway interface {
	CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error)
	UpdateCampaign(context.Context, string, UpdateCampaignInput) error
	UpdateCampaignAIBinding(context.Context, string, UpdateCampaignAIBindingInput) error
	CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error)
	UpdateCharacter(context.Context, string, string, UpdateCharacterInput) error
	DeleteCharacter(context.Context, string, string) error
	SetCharacterController(context.Context, string, string, string) error
	ClaimCharacterControl(context.Context, string, string) error
	ReleaseCharacterControl(context.Context, string, string) error
	CreateParticipant(context.Context, string, CreateParticipantInput) (CreateParticipantResult, error)
	UpdateParticipant(context.Context, string, UpdateParticipantInput) error
	StartSession(context.Context, string, StartSessionInput) error
	EndSession(context.Context, string, EndSessionInput) error
	CreateInvite(context.Context, string, CreateInviteInput) error
	RevokeInvite(context.Context, string, RevokeInviteInput) error
	ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error
	ResetCharacterCreationWorkflow(context.Context, string, string) error
}

// CampaignGateway loads campaign summaries and applies workspace mutations.
type CampaignGateway interface {
	campaignReadGateway
	campaignMutationGateway
}
