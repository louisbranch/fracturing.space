package app

import (
	"context"

	"golang.org/x/text/language"
)

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

func (s service) StartSession(ctx context.Context, campaignID string, input StartSessionInput) error {
	return s.startSession(ctx, campaignID, input)
}

func (s service) EndSession(ctx context.Context, campaignID string, input EndSessionInput) error {
	return s.endSession(ctx, campaignID, input)
}

func (s service) CreateCharacter(ctx context.Context, campaignID string, input CreateCharacterInput) (CreateCharacterResult, error) {
	return s.createCharacter(ctx, campaignID, input)
}

func (s service) CreateInvite(ctx context.Context, campaignID string, input CreateInviteInput) error {
	return s.createInvite(ctx, campaignID, input)
}

func (s service) RevokeInvite(ctx context.Context, campaignID string, input RevokeInviteInput) error {
	return s.revokeInvite(ctx, campaignID, input)
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
