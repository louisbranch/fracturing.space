package app

import (
	"context"
	"sort"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

type campaignReadGateway interface {
	ListCampaigns(context.Context) ([]CampaignSummary, error)
	CampaignName(context.Context, string) (string, error)
	CampaignWorkspace(context.Context, string) (CampaignWorkspace, error)
	CampaignParticipants(context.Context, string) ([]CampaignParticipant, error)
	CampaignCharacters(context.Context, string) ([]CampaignCharacter, error)
	CampaignSessions(context.Context, string) ([]CampaignSession, error)
	CampaignInvites(context.Context, string) ([]CampaignInvite, error)
	CharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	CharacterCreationCatalog(context.Context, language.Tag) (CampaignCharacterCreationCatalog, error)
	CharacterCreationProfile(context.Context, string, string) (CampaignCharacterCreationProfile, error)
}

type campaignMutationGateway interface {
	CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error)
	CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error)
	StartSession(context.Context, string) error
	EndSession(context.Context, string) error
	UpdateParticipants(context.Context, string) error
	UpdateCharacter(context.Context, string) error
	ControlCharacter(context.Context, string) error
	CreateInvite(context.Context, string) error
	RevokeInvite(context.Context, string) error
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
	CampaignCharacters(context.Context, string) ([]CampaignCharacter, error)
	CampaignSessions(context.Context, string) ([]CampaignSession, error)
	CampaignInvites(context.Context, string) ([]CampaignInvite, error)
	StartSession(context.Context, string) error
	EndSession(context.Context, string) error
	UpdateParticipants(context.Context, string) error
	CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error)
	UpdateCharacter(context.Context, string) error
	ControlCharacter(context.Context, string) error
	CreateInvite(context.Context, string) error
	RevokeInvite(context.Context, string) error
	ResolveWorkflow(string) CharacterCreationWorkflow
	CampaignCharacterCreation(context.Context, string, string, language.Tag, CharacterCreationWorkflow) (CampaignCharacterCreation, error)
	CampaignCharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error
	ResetCharacterCreationWorkflow(context.Context, string, string) error
}

type service struct {
	readGateway     campaignReadGateway
	mutationGateway campaignMutationGateway
	authzGateway    campaignAuthzGateway
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

func newService(gateway CampaignGateway) service {
	return newServiceWithWorkflows(gateway, nil)
}

func newServiceWithWorkflows(gateway CampaignGateway, workflows map[string]CharacterCreationWorkflow) service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	var authz campaignAuthzGateway
	if checker, ok := gateway.(campaignAuthzGateway); ok {
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

func (s service) listCampaigns(ctx context.Context) ([]CampaignSummary, error) {
	items, err := s.readGateway.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []CampaignSummary{}, nil
	}
	sorted := append([]CampaignSummary(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].CreatedAtUnixNano > sorted[j].CreatedAtUnixNano
	})
	return sorted, nil
}

func (s service) createCampaign(ctx context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	if strings.TrimSpace(input.Name) == "" {
		return CreateCampaignResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_name_is_required", "campaign name is required")
	}
	created, err := s.mutationGateway.CreateCampaign(ctx, input)
	if err != nil {
		return CreateCampaignResult{}, err
	}
	if strings.TrimSpace(created.CampaignID) == "" {
		return CreateCampaignResult{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.created_campaign_id_was_empty", "created campaign id was empty")
	}
	return created, nil
}

func (s service) campaignName(ctx context.Context, campaignID string) string {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return ""
	}
	name, err := s.readGateway.CampaignName(ctx, campaignID)
	if err != nil {
		return campaignID
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return campaignID
	}
	return name
}

func (s service) campaignWorkspace(ctx context.Context, campaignID string) (CampaignWorkspace, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignWorkspace{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	workspace, err := s.readGateway.CampaignWorkspace(ctx, campaignID)
	if err != nil {
		return CampaignWorkspace{}, err
	}
	workspace.ID = campaignID
	workspace.Name = strings.TrimSpace(workspace.Name)
	if workspace.Name == "" {
		workspace.Name = campaignID
	}
	workspace.Theme = strings.TrimSpace(workspace.Theme)
	workspace.System = strings.TrimSpace(workspace.System)
	if workspace.System == "" {
		workspace.System = "Unspecified"
	}
	workspace.GMMode = strings.TrimSpace(workspace.GMMode)
	if workspace.GMMode == "" {
		workspace.GMMode = "Unspecified"
	}
	workspace.Status = strings.TrimSpace(workspace.Status)
	if workspace.Status == "" {
		workspace.Status = "Unspecified"
	}
	workspace.Locale = strings.TrimSpace(workspace.Locale)
	if workspace.Locale == "" {
		workspace.Locale = "Unspecified"
	}
	workspace.ParticipantCount = strings.TrimSpace(workspace.ParticipantCount)
	if workspace.ParticipantCount == "" {
		workspace.ParticipantCount = "0"
	}
	workspace.CharacterCount = strings.TrimSpace(workspace.CharacterCount)
	if workspace.CharacterCount == "" {
		workspace.CharacterCount = "0"
	}
	workspace.Intent = strings.TrimSpace(workspace.Intent)
	if workspace.Intent == "" {
		workspace.Intent = "Unspecified"
	}
	workspace.AccessPolicy = strings.TrimSpace(workspace.AccessPolicy)
	if workspace.AccessPolicy == "" {
		workspace.AccessPolicy = "Unspecified"
	}
	workspace.CoverImageURL = strings.TrimSpace(workspace.CoverImageURL)
	if workspace.CoverImageURL == "" {
		workspace.CoverImageURL = campaignCoverImageURL("", campaignID, "", "")
	}
	return workspace, nil
}

func (s service) startSession(ctx context.Context, campaignID string) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageSession); err != nil {
		return err
	}
	return s.mutationGateway.StartSession(ctx, campaignID)
}

func (s service) endSession(ctx context.Context, campaignID string) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageSession); err != nil {
		return err
	}
	return s.mutationGateway.EndSession(ctx, campaignID)
}

func (s service) updateParticipants(ctx context.Context, campaignID string) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageParticipant); err != nil {
		return err
	}
	return s.mutationGateway.UpdateParticipants(ctx, campaignID)
}

func (s service) updateCharacter(ctx context.Context, campaignID string) error {
	if err := s.requirePolicy(ctx, campaignID, policyMutateCharacter); err != nil {
		return err
	}
	return s.mutationGateway.UpdateCharacter(ctx, campaignID)
}

func (s service) controlCharacter(ctx context.Context, campaignID string) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageCharacter); err != nil {
		return err
	}
	return s.mutationGateway.ControlCharacter(ctx, campaignID)
}

func (s service) createInvite(ctx context.Context, campaignID string) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageInvite); err != nil {
		return err
	}
	return s.mutationGateway.CreateInvite(ctx, campaignID)
}

func (s service) revokeInvite(ctx context.Context, campaignID string) error {
	if err := s.requirePolicy(ctx, campaignID, policyManageInvite); err != nil {
		return err
	}
	return s.mutationGateway.RevokeInvite(ctx, campaignID)
}

func (s service) createCharacter(ctx context.Context, campaignID string, input CreateCharacterInput) (CreateCharacterResult, error) {
	if strings.TrimSpace(input.Name) == "" {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_name_is_required", "character name is required")
	}
	if input.Kind == CharacterKindUnspecified {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_kind_value_is_invalid", "character kind value is invalid")
	}
	if err := s.requirePolicy(ctx, campaignID, policyMutateCharacter); err != nil {
		return CreateCharacterResult{}, err
	}
	created, err := s.mutationGateway.CreateCharacter(ctx, campaignID, input)
	if err != nil {
		return CreateCharacterResult{}, err
	}
	if strings.TrimSpace(created.CharacterID) == "" {
		return CreateCharacterResult{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.created_character_id_was_empty", "created character id was empty")
	}
	return created, nil
}

func (s service) ListCampaigns(ctx context.Context) ([]CampaignSummary, error) {
	return s.listCampaigns(ctx)
}

func (s service) CreateCampaign(ctx context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	return s.createCampaign(ctx, input)
}

func (s service) CampaignName(ctx context.Context, campaignID string) string {
	return s.campaignName(ctx, campaignID)
}

func (s service) CampaignWorkspace(ctx context.Context, campaignID string) (CampaignWorkspace, error) {
	return s.campaignWorkspace(ctx, campaignID)
}

func (s service) CampaignParticipants(ctx context.Context, campaignID string) ([]CampaignParticipant, error) {
	return s.campaignParticipants(ctx, campaignID)
}

func (s service) CampaignCharacters(ctx context.Context, campaignID string) ([]CampaignCharacter, error) {
	return s.campaignCharacters(ctx, campaignID)
}

func (s service) CampaignSessions(ctx context.Context, campaignID string) ([]CampaignSession, error) {
	return s.campaignSessions(ctx, campaignID)
}

func (s service) CampaignInvites(ctx context.Context, campaignID string) ([]CampaignInvite, error) {
	return s.campaignInvites(ctx, campaignID)
}

func (s service) StartSession(ctx context.Context, campaignID string) error {
	return s.startSession(ctx, campaignID)
}

func (s service) EndSession(ctx context.Context, campaignID string) error {
	return s.endSession(ctx, campaignID)
}

func (s service) UpdateParticipants(ctx context.Context, campaignID string) error {
	return s.updateParticipants(ctx, campaignID)
}

func (s service) CreateCharacter(ctx context.Context, campaignID string, input CreateCharacterInput) (CreateCharacterResult, error) {
	return s.createCharacter(ctx, campaignID, input)
}

func (s service) UpdateCharacter(ctx context.Context, campaignID string) error {
	return s.updateCharacter(ctx, campaignID)
}

func (s service) ControlCharacter(ctx context.Context, campaignID string) error {
	return s.controlCharacter(ctx, campaignID)
}

func (s service) CreateInvite(ctx context.Context, campaignID string) error {
	return s.createInvite(ctx, campaignID)
}

func (s service) RevokeInvite(ctx context.Context, campaignID string) error {
	return s.revokeInvite(ctx, campaignID)
}

func (s service) ResolveWorkflow(system string) CharacterCreationWorkflow {
	return s.resolveWorkflow(system)
}

func (s service) CampaignCharacterCreation(ctx context.Context, campaignID string, characterID string, locale language.Tag, workflow CharacterCreationWorkflow) (CampaignCharacterCreation, error) {
	return s.campaignCharacterCreation(ctx, campaignID, characterID, locale, workflow)
}

func (s service) CampaignCharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProgress, error) {
	return s.campaignCharacterCreationProgress(ctx, campaignID, characterID)
}

func (s service) ApplyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *CampaignCharacterCreationStepInput) error {
	return s.applyCharacterCreationStep(ctx, campaignID, characterID, step)
}

func (s service) ResetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	return s.resetCharacterCreationWorkflow(ctx, campaignID, characterID)
}
