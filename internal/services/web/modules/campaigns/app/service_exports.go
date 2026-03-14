package app

import (
	"context"

	"golang.org/x/text/language"
)

// ListCampaigns returns the package view collection for this workflow.
func (s service) ListCampaigns(ctx context.Context) ([]CampaignSummary, error) {
	return s.listCampaigns(ctx)
}

// CreateCampaign executes package-scoped creation behavior for this flow.
func (s service) CreateCampaign(ctx context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	return s.createCampaign(ctx, input)
}

// CampaignName centralizes this web behavior in one helper seam.
func (s service) CampaignName(ctx context.Context, campaignID string) string {
	return s.campaignName(ctx, campaignID)
}

// CampaignWorkspace centralizes this web behavior in one helper seam.
func (s service) CampaignWorkspace(ctx context.Context, campaignID string) (CampaignWorkspace, error) {
	return s.campaignWorkspace(ctx, campaignID)
}

// CampaignParticipants centralizes this web behavior in one helper seam.
func (s service) CampaignParticipants(ctx context.Context, campaignID string) ([]CampaignParticipant, error) {
	return s.campaignParticipants(ctx, campaignID)
}

// CampaignParticipantEditor centralizes this web behavior in one helper seam.
func (s service) CampaignParticipantEditor(ctx context.Context, campaignID string, participantID string) (CampaignParticipantEditor, error) {
	return s.campaignParticipantEditor(ctx, campaignID, participantID)
}

// CampaignAIBindingEditor centralizes this web behavior in one helper seam.
func (s service) CampaignAIBindingEditor(ctx context.Context, campaignID string, currentAIAgentID string) (CampaignAIBindingEditor, error) {
	return s.campaignAIBindingEditor(ctx, campaignID, currentAIAgentID)
}

// CampaignCharacters centralizes this web behavior in one helper seam.
func (s service) CampaignCharacters(ctx context.Context, campaignID string) ([]CampaignCharacter, error) {
	return s.campaignCharacters(ctx, campaignID)
}

// CampaignCharacterEditor centralizes this web behavior in one helper seam.
func (s service) CampaignCharacterEditor(ctx context.Context, campaignID string, characterID string) (CampaignCharacterEditor, error) {
	return s.campaignCharacterEditor(ctx, campaignID, characterID)
}

// CampaignCharacterControl centralizes this web behavior in one helper seam.
func (s service) CampaignCharacterControl(ctx context.Context, campaignID string, characterID string, userID string) (CampaignCharacterControl, error) {
	return s.campaignCharacterControl(ctx, campaignID, characterID, userID)
}

// CampaignSessions centralizes this web behavior in one helper seam.
func (s service) CampaignSessions(ctx context.Context, campaignID string) ([]CampaignSession, error) {
	return s.campaignSessions(ctx, campaignID)
}

// CampaignSessionReadiness centralizes this web behavior in one helper seam.
func (s service) CampaignSessionReadiness(ctx context.Context, campaignID string, locale language.Tag) (CampaignSessionReadiness, error) {
	return s.campaignSessionReadiness(ctx, campaignID, locale)
}

// CampaignInvites centralizes this web behavior in one helper seam.
func (s service) CampaignInvites(ctx context.Context, campaignID string) ([]CampaignInvite, error) {
	return s.campaignInvites(ctx, campaignID)
}

// RequireManageCampaign enforces owner/manager campaign access.
func (s service) RequireManageCampaign(ctx context.Context, campaignID string) error {
	return s.requireManageCampaign(ctx, campaignID)
}

// RequireManageParticipants enforces owner/manager participant governance access.
func (s service) RequireManageParticipants(ctx context.Context, campaignID string) error {
	return s.requireManageParticipants(ctx, campaignID)
}

// RequireMutateCharacters enforces character-mutation access.
func (s service) RequireMutateCharacters(ctx context.Context, campaignID string) error {
	return s.requireMutateCharacters(ctx, campaignID)
}

// UpdateCampaign applies this package workflow transition.
func (s service) UpdateCampaign(ctx context.Context, campaignID string, input UpdateCampaignInput) error {
	return s.updateCampaign(ctx, campaignID, input)
}

// UpdateCampaignAIBinding applies this package workflow transition.
func (s service) UpdateCampaignAIBinding(ctx context.Context, campaignID string, input UpdateCampaignAIBindingInput) error {
	return s.updateCampaignAIBinding(ctx, campaignID, input)
}

// StartSession applies this package workflow transition.
func (s service) StartSession(ctx context.Context, campaignID string, input StartSessionInput) error {
	return s.startSession(ctx, campaignID, input)
}

// EndSession applies this package workflow transition.
func (s service) EndSession(ctx context.Context, campaignID string, input EndSessionInput) error {
	return s.endSession(ctx, campaignID, input)
}

// CreateCharacter executes package-scoped creation behavior for this flow.
func (s service) CreateCharacter(ctx context.Context, campaignID string, input CreateCharacterInput) (CreateCharacterResult, error) {
	return s.createCharacter(ctx, campaignID, input)
}

// UpdateCharacter applies this package workflow transition.
func (s service) UpdateCharacter(ctx context.Context, campaignID string, characterID string, input UpdateCharacterInput) error {
	return s.updateCharacter(ctx, campaignID, characterID, input)
}

// DeleteCharacter applies this package workflow transition.
func (s service) DeleteCharacter(ctx context.Context, campaignID string, characterID string) error {
	return s.deleteCharacter(ctx, campaignID, characterID)
}

// SetCharacterController applies this package workflow transition.
func (s service) SetCharacterController(ctx context.Context, campaignID string, characterID string, participantID string) error {
	return s.setCharacterController(ctx, campaignID, characterID, participantID)
}

// ClaimCharacterControl applies this package workflow transition.
func (s service) ClaimCharacterControl(ctx context.Context, campaignID string, characterID string, userID string) error {
	return s.claimCharacterControl(ctx, campaignID, characterID, userID)
}

// ReleaseCharacterControl applies this package workflow transition.
func (s service) ReleaseCharacterControl(ctx context.Context, campaignID string, characterID string, userID string) error {
	return s.releaseCharacterControl(ctx, campaignID, characterID, userID)
}

// UpdateParticipant applies this package workflow transition.
func (s service) UpdateParticipant(ctx context.Context, campaignID string, input UpdateParticipantInput) error {
	return s.updateParticipant(ctx, campaignID, input)
}

// CreateInvite executes package-scoped creation behavior for this flow.
func (s service) CreateInvite(ctx context.Context, campaignID string, input CreateInviteInput) error {
	return s.createInvite(ctx, campaignID, input)
}

// RevokeInvite applies this package workflow transition.
func (s service) RevokeInvite(ctx context.Context, campaignID string, input RevokeInviteInput) error {
	return s.revokeInvite(ctx, campaignID, input)
}

// ResolveWorkflow resolves request-scoped values needed by this package.
func (s service) ResolveWorkflow(system string) CharacterCreationWorkflow {
	return s.resolveWorkflow(system)
}

// CampaignCharacterCreation centralizes this web behavior in one helper seam.
func (s service) CampaignCharacterCreation(ctx context.Context, campaignID string, characterID string, locale language.Tag, workflow CharacterCreationWorkflow) (CampaignCharacterCreation, error) {
	return s.campaignCharacterCreation(ctx, campaignID, characterID, locale, workflow)
}

// CampaignCharacterCreationProgress centralizes this web behavior in one helper seam.
func (s service) CampaignCharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProgress, error) {
	return s.campaignCharacterCreationProgress(ctx, campaignID, characterID)
}

// ApplyCharacterCreationStep applies this package workflow transition.
func (s service) ApplyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *CampaignCharacterCreationStepInput) error {
	return s.applyCharacterCreationStep(ctx, campaignID, characterID, step)
}

// ResetCharacterCreationWorkflow applies this package workflow transition.
func (s service) ResetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	return s.resetCharacterCreationWorkflow(ctx, campaignID, characterID)
}
