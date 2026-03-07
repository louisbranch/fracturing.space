package app

import (
	"context"
	"strings"

	"golang.org/x/text/language"
)

// campaignReadGateway defines an internal contract used at this web package boundary.
type campaignReadGateway interface {
	ListCampaigns(context.Context) ([]CampaignSummary, error)
	CampaignName(context.Context, string) (string, error)
	CampaignWorkspace(context.Context, string) (CampaignWorkspace, error)
	CampaignParticipants(context.Context, string) ([]CampaignParticipant, error)
	CampaignParticipant(context.Context, string, string) (CampaignParticipant, error)
	CampaignCharacters(context.Context, string) ([]CampaignCharacter, error)
	CampaignSessions(context.Context, string) ([]CampaignSession, error)
	CampaignSessionReadiness(context.Context, string, language.Tag) (CampaignSessionReadiness, error)
	CampaignInvites(context.Context, string) ([]CampaignInvite, error)
	CharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	CharacterCreationCatalog(context.Context, language.Tag) (CampaignCharacterCreationCatalog, error)
	CharacterCreationProfile(context.Context, string, string) (CampaignCharacterCreationProfile, error)
}

// campaignMutationGateway defines an internal contract used at this web package boundary.
type campaignMutationGateway interface {
	CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error)
	UpdateCampaign(context.Context, string, UpdateCampaignInput) error
	CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error)
	UpdateCharacter(context.Context, string, string, UpdateCharacterInput) error
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

// Service orchestrates campaign workspace reads and mutations.
type Service interface {
	ListCampaigns(context.Context) ([]CampaignSummary, error)
	CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error)
	CampaignName(context.Context, string) string
	CampaignWorkspace(context.Context, string) (CampaignWorkspace, error)
	CampaignParticipants(context.Context, string) ([]CampaignParticipant, error)
	CampaignParticipantEditor(context.Context, string, string) (CampaignParticipantEditor, error)
	CampaignCharacters(context.Context, string) ([]CampaignCharacter, error)
	CampaignSessions(context.Context, string) ([]CampaignSession, error)
	CampaignSessionReadiness(context.Context, string, language.Tag) (CampaignSessionReadiness, error)
	CampaignInvites(context.Context, string) ([]CampaignInvite, error)
	RequireManageCampaign(context.Context, string) error
	UpdateCampaign(context.Context, string, UpdateCampaignInput) error
	StartSession(context.Context, string, StartSessionInput) error
	EndSession(context.Context, string, EndSessionInput) error
	CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error)
	UpdateCharacter(context.Context, string, string, UpdateCharacterInput) error
	UpdateParticipant(context.Context, string, UpdateParticipantInput) error
	CreateInvite(context.Context, string, CreateInviteInput) error
	RevokeInvite(context.Context, string, RevokeInviteInput) error
	ResolveWorkflow(string) CharacterCreationWorkflow
	CampaignCharacterCreation(context.Context, string, string, language.Tag, CharacterCreationWorkflow) (CampaignCharacterCreation, error)
	CampaignCharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error
	ResetCharacterCreationWorkflow(context.Context, string, string) error
}

// service defines an internal contract used at this web package boundary.
type service struct {
	readGateway     campaignReadGateway
	mutationGateway campaignMutationGateway
	authzGateway    AuthzGateway
	workflows       map[string]CharacterCreationWorkflow
}

// NewService constructs a service with default workflows.
func NewService(gateway CampaignGateway) Service {
	return newService(gateway)
}

// NewServiceWithWorkflows constructs a service with explicit workflow map.
func NewServiceWithWorkflows(gateway CampaignGateway, workflows map[string]CharacterCreationWorkflow) Service {
	return newServiceWithWorkflows(gateway, workflows)
}

// IsGatewayHealthy reports whether the gateway is present and operational.
func IsGatewayHealthy(gateway CampaignGateway) bool {
	if gateway == nil {
		return false
	}
	_, unavailable := gateway.(unavailableGateway)
	return !unavailable
}

// newService builds package wiring for this web seam.
func newService(gateway CampaignGateway) service {
	return newServiceWithWorkflows(gateway, nil)
}

// newServiceWithWorkflows builds package wiring for this web seam.
func newServiceWithWorkflows(gateway CampaignGateway, workflows map[string]CharacterCreationWorkflow) service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	var authz AuthzGateway
	if checker, ok := gateway.(AuthzGateway); ok {
		authz = checker
	}
	return service{
		readGateway:     gateway,
		mutationGateway: gateway,
		authzGateway:    authz,
		workflows:       workflows,
	}
}

// resolveWorkflow returns the workflow implementation for the given system, or nil.
func (s service) resolveWorkflow(system string) CharacterCreationWorkflow {
	if s.workflows == nil {
		return nil
	}
	return s.workflows[strings.ToLower(strings.TrimSpace(system))]
}
