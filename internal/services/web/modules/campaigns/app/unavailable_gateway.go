package app

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

// unavailableGateway defines an internal contract used at this web package boundary.
type unavailableGateway struct{}

// NewUnavailableGateway returns a gateway that fails closed with unavailable errors.
func NewUnavailableGateway() CampaignGateway {
	return unavailableGateway{}
}

// ListCampaigns returns the package view collection for this workflow.
func (unavailableGateway) ListCampaigns(context.Context) ([]CampaignSummary, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CampaignName centralizes this web behavior in one helper seam.
func (unavailableGateway) CampaignName(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CampaignWorkspace centralizes this web behavior in one helper seam.
func (unavailableGateway) CampaignWorkspace(context.Context, string) (CampaignWorkspace, error) {
	return CampaignWorkspace{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CampaignAIAgents centralizes this web behavior in one helper seam.
func (unavailableGateway) CampaignAIAgents(context.Context) ([]CampaignAIAgentOption, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaign ai agents are not configured")
}

// CampaignParticipants centralizes this web behavior in one helper seam.
func (unavailableGateway) CampaignParticipants(context.Context, string) ([]CampaignParticipant, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CampaignParticipant centralizes this web behavior in one helper seam.
func (unavailableGateway) CampaignParticipant(context.Context, string, string) (CampaignParticipant, error) {
	return CampaignParticipant{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CampaignCharacters centralizes this web behavior in one helper seam.
func (unavailableGateway) CampaignCharacters(context.Context, string) ([]CampaignCharacter, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CampaignSessions centralizes this web behavior in one helper seam.
func (unavailableGateway) CampaignSessions(context.Context, string) ([]CampaignSession, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CampaignSessionReadiness centralizes this web behavior in one helper seam.
func (unavailableGateway) CampaignSessionReadiness(context.Context, string, language.Tag) (CampaignSessionReadiness, error) {
	return CampaignSessionReadiness{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CampaignInvites centralizes this web behavior in one helper seam.
func (unavailableGateway) CampaignInvites(context.Context, string) ([]CampaignInvite, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CharacterCreationProgress centralizes this web behavior in one helper seam.
func (unavailableGateway) CharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error) {
	return CampaignCharacterCreationProgress{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CharacterCreationCatalog centralizes this web behavior in one helper seam.
func (unavailableGateway) CharacterCreationCatalog(context.Context, language.Tag) (CampaignCharacterCreationCatalog, error) {
	return CampaignCharacterCreationCatalog{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CharacterCreationProfile centralizes this web behavior in one helper seam.
func (unavailableGateway) CharacterCreationProfile(context.Context, string, string) (CampaignCharacterCreationProfile, error) {
	return CampaignCharacterCreationProfile{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CreateCampaign executes package-scoped creation behavior for this flow.
func (unavailableGateway) CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error) {
	return CreateCampaignResult{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// UpdateCampaign applies this package workflow transition.
func (unavailableGateway) UpdateCampaign(context.Context, string, UpdateCampaignInput) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// UpdateCampaignAIBinding applies this package workflow transition.
func (unavailableGateway) UpdateCampaignAIBinding(context.Context, string, UpdateCampaignAIBindingInput) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// StartSession applies this package workflow transition.
func (unavailableGateway) StartSession(context.Context, string, StartSessionInput) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// EndSession applies this package workflow transition.
func (unavailableGateway) EndSession(context.Context, string, EndSessionInput) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CreateCharacter executes package-scoped creation behavior for this flow.
func (unavailableGateway) CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error) {
	return CreateCharacterResult{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// UpdateCharacter applies this package workflow transition.
func (unavailableGateway) UpdateCharacter(context.Context, string, string, UpdateCharacterInput) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// DeleteCharacter applies this package workflow transition.
func (unavailableGateway) DeleteCharacter(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// SetCharacterController applies this package workflow transition.
func (unavailableGateway) SetCharacterController(context.Context, string, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// ClaimCharacterControl applies this package workflow transition.
func (unavailableGateway) ClaimCharacterControl(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// ReleaseCharacterControl applies this package workflow transition.
func (unavailableGateway) ReleaseCharacterControl(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// UpdateParticipant applies this package workflow transition.
func (unavailableGateway) UpdateParticipant(context.Context, string, UpdateParticipantInput) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// CreateInvite executes package-scoped creation behavior for this flow.
func (unavailableGateway) CreateInvite(context.Context, string, CreateInviteInput) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// RevokeInvite applies this package workflow transition.
func (unavailableGateway) RevokeInvite(context.Context, string, RevokeInviteInput) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// ApplyCharacterCreationStep applies this package workflow transition.
func (unavailableGateway) ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

// ResetCharacterCreationWorkflow applies this package workflow transition.
func (unavailableGateway) ResetCharacterCreationWorkflow(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}
