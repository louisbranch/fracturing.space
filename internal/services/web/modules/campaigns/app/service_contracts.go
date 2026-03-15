package app

import (
	"context"

	"golang.org/x/text/language"
)

// CampaignCatalogService exposes campaign catalog reads and mutations.
type CampaignCatalogService interface {
	ListCampaigns(context.Context) ([]CampaignSummary, error)
	CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error)
}

// CampaignWorkspaceService exposes campaign workspace reads used by transport.
type CampaignWorkspaceService interface {
	CampaignName(context.Context, string) string
	CampaignWorkspace(context.Context, string) (CampaignWorkspace, error)
}

// CampaignGameService exposes game-surface reads used by transport.
type CampaignGameService interface {
	CampaignGameSurface(context.Context, string) (CampaignGameSurface, error)
}

// CampaignParticipantReadService exposes participant-focused reads and editor
// state.
type CampaignParticipantReadService interface {
	CampaignParticipants(context.Context, string) ([]CampaignParticipant, error)
	CampaignParticipantCreator(context.Context, string) (CampaignParticipantCreator, error)
	CampaignParticipantEditor(context.Context, string, string) (CampaignParticipantEditor, error)
}

// CampaignParticipantMutationService exposes participant create/update
// mutations.
type CampaignParticipantMutationService interface {
	CreateParticipant(context.Context, string, CreateParticipantInput) (CreateParticipantResult, error)
	UpdateParticipant(context.Context, string, UpdateParticipantInput) error
}

// CampaignAutomationReadService exposes campaign-level AI automation reads.
type CampaignAutomationReadService interface {
	CampaignAIBindingSummary(context.Context, string, string, string) (CampaignAIBindingSummary, error)
	CampaignAIBindingSettings(context.Context, string, string) (CampaignAIBindingSettings, error)
}

// CampaignAutomationMutationService exposes campaign-level AI automation
// mutations.
type CampaignAutomationMutationService interface {
	UpdateCampaignAIBinding(context.Context, string, UpdateCampaignAIBindingInput) error
}

// CampaignCharacterReadService exposes character list/entity/editor reads.
type CampaignCharacterReadService interface {
	CampaignCharacters(context.Context, string, CharacterReadContext) ([]CampaignCharacter, error)
	CampaignCharacter(context.Context, string, string, CharacterReadContext) (CampaignCharacter, error)
	CampaignCharacterEditor(context.Context, string, string, CharacterReadContext) (CampaignCharacterEditor, error)
}

// CampaignCharacterControlService exposes character-control detail state and
// control mutations.
type CampaignCharacterControlService interface {
	CampaignCharacterControl(context.Context, string, string, string, CharacterReadContext) (CampaignCharacterControl, error)
	SetCharacterController(context.Context, string, string, string) error
	ClaimCharacterControl(context.Context, string, string, string) error
	ReleaseCharacterControl(context.Context, string, string, string) error
}

// CampaignCharacterMutationService exposes character create/update/delete
// mutations.
type CampaignCharacterMutationService interface {
	CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error)
	UpdateCharacter(context.Context, string, string, UpdateCharacterInput) error
	DeleteCharacter(context.Context, string, string) error
}

// CampaignSessionReadService exposes session list/readiness reads.
type CampaignSessionReadService interface {
	CampaignSessions(context.Context, string) ([]CampaignSession, error)
	CampaignSessionReadiness(context.Context, string, language.Tag) (CampaignSessionReadiness, error)
}

// CampaignSessionMutationService exposes session lifecycle mutations.
type CampaignSessionMutationService interface {
	StartSession(context.Context, string, StartSessionInput) error
	EndSession(context.Context, string, EndSessionInput) error
}

// CampaignInviteReadService exposes invite-focused reads and search.
type CampaignInviteReadService interface {
	CampaignInvites(context.Context, string) ([]CampaignInvite, error)
	SearchInviteUsers(context.Context, string, SearchInviteUsersInput) ([]InviteUserSearchResult, error)
}

// CampaignInviteMutationService exposes invite create/revoke mutations.
type CampaignInviteMutationService interface {
	CreateInvite(context.Context, string, CreateInviteInput) error
	RevokeInvite(context.Context, string, RevokeInviteInput) error
}

// CampaignConfigurationService exposes campaign-level settings mutations.
type CampaignConfigurationService interface {
	UpdateCampaign(context.Context, string, UpdateCampaignInput) error
}

// CampaignAuthorizationService exposes transport-facing authorization checks.
type CampaignAuthorizationService interface {
	RequireManageCampaign(context.Context, string) error
	RequireManageParticipants(context.Context, string) error
	RequireManageInvites(context.Context, string) error
	RequireMutateCharacters(context.Context, string) error
}

// CampaignCharacterCreationPageService exposes character-creation page reads.
type CampaignCharacterCreationPageService interface {
	CampaignCharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	CampaignCharacterCreationCatalog(context.Context, language.Tag) (CampaignCharacterCreationCatalog, error)
	CampaignCharacterCreationProfile(context.Context, string, string) (CampaignCharacterCreationProfile, error)
}

// CampaignCharacterCreationMutationService exposes character-creation workflow
// progress reads and mutations.
type CampaignCharacterCreationMutationService interface {
	CampaignCharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error
	ResetCharacterCreationWorkflow(context.Context, string, string) error
}
