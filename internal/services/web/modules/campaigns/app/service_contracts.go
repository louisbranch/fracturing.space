package app

import (
	"context"

	"golang.org/x/text/language"
)

// Service orchestrates campaign workspace reads and mutations.
type Service interface {
	ListCampaigns(context.Context) ([]CampaignSummary, error)
	CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error)
	CampaignName(context.Context, string) string
	CampaignWorkspace(context.Context, string) (CampaignWorkspace, error)
	CampaignGameSurface(context.Context, string) (CampaignGameSurface, error)
	CampaignParticipants(context.Context, string) ([]CampaignParticipant, error)
	CampaignParticipantCreator(context.Context, string) (CampaignParticipantCreator, error)
	CampaignParticipantEditor(context.Context, string, string) (CampaignParticipantEditor, error)
	CampaignAIBindingEditor(context.Context, string, string) (CampaignAIBindingEditor, error)
	CampaignCharacters(context.Context, string, CampaignCharactersReadOptions) ([]CampaignCharacter, error)
	CampaignCharacterEditor(context.Context, string, string) (CampaignCharacterEditor, error)
	CampaignCharacterControl(context.Context, string, string, string) (CampaignCharacterControl, error)
	CampaignSessions(context.Context, string) ([]CampaignSession, error)
	CampaignSessionReadiness(context.Context, string, language.Tag) (CampaignSessionReadiness, error)
	CampaignInvites(context.Context, string) ([]CampaignInvite, error)
	SearchInviteUsers(context.Context, string, SearchInviteUsersInput) ([]InviteUserSearchResult, error)
	RequireManageCampaign(context.Context, string) error
	RequireManageParticipants(context.Context, string) error
	RequireManageInvites(context.Context, string) error
	RequireMutateCharacters(context.Context, string) error
	UpdateCampaign(context.Context, string, UpdateCampaignInput) error
	UpdateCampaignAIBinding(context.Context, string, UpdateCampaignAIBindingInput) error
	StartSession(context.Context, string, StartSessionInput) error
	EndSession(context.Context, string, EndSessionInput) error
	CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error)
	UpdateCharacter(context.Context, string, string, UpdateCharacterInput) error
	DeleteCharacter(context.Context, string, string) error
	SetCharacterController(context.Context, string, string, string) error
	ClaimCharacterControl(context.Context, string, string, string) error
	ReleaseCharacterControl(context.Context, string, string, string) error
	CreateParticipant(context.Context, string, CreateParticipantInput) (CreateParticipantResult, error)
	UpdateParticipant(context.Context, string, UpdateParticipantInput) error
	CreateInvite(context.Context, string, CreateInviteInput) error
	RevokeInvite(context.Context, string, RevokeInviteInput) error
	CampaignCharacterCreationData(context.Context, string, string, language.Tag) (CampaignCharacterCreationData, error)
	CampaignCharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error
	ResetCharacterCreationWorkflow(context.Context, string, string) error
}

// ServiceConfig keeps constructor dependencies explicit by capability.
type ServiceConfig struct {
	ReadGateway     ReadGateway
	MutationGateway MutationGateway
	AuthzGateway    AuthzGateway
}

// service defines an internal contract used at this web package boundary.
type service struct {
	readGateway     ReadGateway
	mutationGateway MutationGateway
	authzGateway    AuthzGateway
}

// NewService constructs a service from explicit read, mutation, and authz seams.
func NewService(config ServiceConfig) Service {
	return newServiceWithConfig(config)
}

// IsGatewayHealthy reports whether the read gateway is present and operational.
func IsGatewayHealthy(readGateway ReadGateway) bool {
	if readGateway == nil {
		return false
	}
	_, unavailable := readGateway.(unavailableGateway)
	return !unavailable
}

// newService keeps package-local tests on a convenient combined-gateway seam.
func newService(gateway CampaignGateway) service {
	if gateway == nil {
		return newServiceWithConfig(ServiceConfig{})
	}
	var authz AuthzGateway
	if checker, ok := gateway.(AuthzGateway); ok {
		authz = checker
	}
	return newServiceWithConfig(ServiceConfig{
		ReadGateway:     gateway,
		MutationGateway: gateway,
		AuthzGateway:    authz,
	})
}

// newServiceWithConfig builds package wiring for this web seam.
func newServiceWithConfig(config ServiceConfig) service {
	readGateway := config.ReadGateway
	if readGateway == nil {
		readGateway = unavailableGateway{}
	}
	mutationGateway := config.MutationGateway
	if mutationGateway == nil {
		mutationGateway = unavailableGateway{}
	}
	return service{
		readGateway:     readGateway,
		mutationGateway: mutationGateway,
		authzGateway:    config.AuthzGateway,
	}
}
