package campaigns

import (
	"context"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"golang.org/x/text/language"
)

// service adapts campaigns/app.Service to the root campaigns handler contract.
type service struct {
	inner campaignapp.Service
}

func newService(gateway CampaignGateway) service {
	return newServiceWithWorkflows(gateway, nil)
}

func newServiceWithWorkflows(gateway CampaignGateway, workflows map[string]CharacterCreationWorkflow) service {
	return service{inner: campaignapp.NewServiceWithWorkflows(gateway, workflows)}
}

func (s service) listCampaigns(ctx context.Context) ([]CampaignSummary, error) {
	return s.inner.ListCampaigns(ctx)
}

func (s service) createCampaign(ctx context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	return s.inner.CreateCampaign(ctx, input)
}

func (s service) campaignName(ctx context.Context, campaignID string) string {
	return s.inner.CampaignName(ctx, campaignID)
}

func (s service) campaignWorkspace(ctx context.Context, campaignID string) (CampaignWorkspace, error) {
	return s.inner.CampaignWorkspace(ctx, campaignID)
}

func (s service) campaignParticipants(ctx context.Context, campaignID string) ([]CampaignParticipant, error) {
	return s.inner.CampaignParticipants(ctx, campaignID)
}

func (s service) campaignCharacters(ctx context.Context, campaignID string) ([]CampaignCharacter, error) {
	return s.inner.CampaignCharacters(ctx, campaignID)
}

func (s service) campaignSessions(ctx context.Context, campaignID string) ([]CampaignSession, error) {
	return s.inner.CampaignSessions(ctx, campaignID)
}

func (s service) campaignInvites(ctx context.Context, campaignID string) ([]CampaignInvite, error) {
	return s.inner.CampaignInvites(ctx, campaignID)
}

func (s service) startSession(ctx context.Context, campaignID string) error {
	return s.inner.StartSession(ctx, campaignID)
}

func (s service) endSession(ctx context.Context, campaignID string) error {
	return s.inner.EndSession(ctx, campaignID)
}

func (s service) updateParticipants(ctx context.Context, campaignID string) error {
	return s.inner.UpdateParticipants(ctx, campaignID)
}

func (s service) createCharacter(ctx context.Context, campaignID string, input CreateCharacterInput) (CreateCharacterResult, error) {
	return s.inner.CreateCharacter(ctx, campaignID, input)
}

func (s service) updateCharacter(ctx context.Context, campaignID string) error {
	return s.inner.UpdateCharacter(ctx, campaignID)
}

func (s service) controlCharacter(ctx context.Context, campaignID string) error {
	return s.inner.ControlCharacter(ctx, campaignID)
}

func (s service) createInvite(ctx context.Context, campaignID string) error {
	return s.inner.CreateInvite(ctx, campaignID)
}

func (s service) revokeInvite(ctx context.Context, campaignID string) error {
	return s.inner.RevokeInvite(ctx, campaignID)
}

func (s service) resolveWorkflow(system string) CharacterCreationWorkflow {
	return s.inner.ResolveWorkflow(system)
}

func (s service) campaignCharacterCreation(ctx context.Context, campaignID string, characterID string, locale language.Tag, workflow CharacterCreationWorkflow) (CampaignCharacterCreation, error) {
	return s.inner.CampaignCharacterCreation(ctx, campaignID, characterID, locale, workflow)
}

func (s service) campaignCharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProgress, error) {
	return s.inner.CampaignCharacterCreationProgress(ctx, campaignID, characterID)
}

func (s service) applyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *CampaignCharacterCreationStepInput) error {
	return s.inner.ApplyCharacterCreationStep(ctx, campaignID, characterID, step)
}

func (s service) resetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	return s.inner.ResetCharacterCreationWorkflow(ctx, campaignID, characterID)
}
