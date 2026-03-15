package app

import (
	"context"

	"golang.org/x/text/language"
)

// CampaignCatalogReadGateway loads campaign list reads for the web service.
type CampaignCatalogReadGateway interface {
	ListCampaigns(context.Context) ([]CampaignSummary, error)
}

// CampaignStarterGateway loads protected starter preview state and launches starter forks.
type CampaignStarterGateway interface {
	StarterPreview(context.Context, string) (CampaignStarterPreview, error)
	LaunchStarter(context.Context, string, LaunchStarterInput) (StarterLaunchResult, error)
}

// CampaignWorkspaceReadGateway loads campaign workspace metadata reads for the
// web service.
type CampaignWorkspaceReadGateway interface {
	CampaignName(context.Context, string) (string, error)
	CampaignWorkspace(context.Context, string) (CampaignWorkspace, error)
}

// CampaignGameReadGateway loads game-surface reads for the web service.
type CampaignGameReadGateway interface {
	CampaignGameSurface(context.Context, string) (CampaignGameSurface, error)
}

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

// CampaignSessionReadGateway loads session reads for the web service.
type CampaignSessionReadGateway interface {
	CampaignSessions(context.Context, string) ([]CampaignSession, error)
	CampaignSessionReadiness(context.Context, string, language.Tag) (CampaignSessionReadiness, error)
}

// CampaignInviteReadGateway loads invite reads for the web service.
type CampaignInviteReadGateway interface {
	CampaignInvites(context.Context, string) ([]CampaignInvite, error)
	SearchInviteUsers(context.Context, SearchInviteUsersInput) ([]InviteUserSearchResult, error)
}

// CampaignAutomationReadGateway loads campaign automation reads for the web
// service.
type CampaignAutomationReadGateway interface {
	CampaignAIAgents(context.Context) ([]CampaignAIAgentOption, error)
}

// CharacterCreationReadGateway loads character-creation reads for the web
// workflow surface.
type CharacterCreationReadGateway interface {
	CharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	CharacterCreationCatalog(context.Context, language.Tag) (CampaignCharacterCreationCatalog, error)
	CharacterCreationProfile(context.Context, string, string) (CampaignCharacterCreationProfile, error)
}

// CampaignCatalogMutationGateway applies campaign catalog mutations for the web service.
type CampaignCatalogMutationGateway interface {
	CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error)
}

// CampaignConfigurationMutationGateway applies campaign-level settings
// mutations for the web service.
type CampaignConfigurationMutationGateway interface {
	UpdateCampaign(context.Context, string, UpdateCampaignInput) error
}

// CampaignAutomationMutationGateway applies campaign-level automation mutations
// for the web service.
type CampaignAutomationMutationGateway interface {
	UpdateCampaignAIBinding(context.Context, string, UpdateCampaignAIBindingInput) error
}

// CampaignCharacterControlMutationGateway applies character-controller
// mutations for the web service.
type CampaignCharacterControlMutationGateway interface {
	SetCharacterController(context.Context, string, string, string) error
	ClaimCharacterControl(context.Context, string, string) error
	ReleaseCharacterControl(context.Context, string, string) error
}

// CampaignCharacterMutationGateway applies character create/update/delete
// mutations for the web service.
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

// CampaignSessionMutationGateway applies session lifecycle mutations for the web service.
type CampaignSessionMutationGateway interface {
	StartSession(context.Context, string, StartSessionInput) error
	EndSession(context.Context, string, EndSessionInput) error
}

// CampaignInviteMutationGateway applies invite mutations for the web service.
type CampaignInviteMutationGateway interface {
	CreateInvite(context.Context, string, CreateInviteInput) error
	RevokeInvite(context.Context, string, RevokeInviteInput) error
}

// CharacterCreationMutationGateway applies character-creation workflow
// mutations for the web service.
type CharacterCreationMutationGateway interface {
	ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error
	ResetCharacterCreationWorkflow(context.Context, string, string) error
}
