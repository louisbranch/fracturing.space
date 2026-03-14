package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

// campaignCharacterCreationData centralizes generic character-creation reads in
// one app-owned helper seam.
func (s service) campaignCharacterCreationData(ctx context.Context, campaignID string, characterID string, locale language.Tag) (CampaignCharacterCreationData, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignCharacterCreationData{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return CampaignCharacterCreationData{}, apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}

	progress, err := s.readGateway.CharacterCreationProgress(ctx, campaignID, characterID)
	if err != nil {
		return CampaignCharacterCreationData{}, err
	}

	catalog, err := s.readGateway.CharacterCreationCatalog(ctx, locale)
	if err != nil {
		return CampaignCharacterCreationData{}, err
	}

	profile, err := s.readGateway.CharacterCreationProfile(ctx, campaignID, characterID)
	if err != nil {
		return CampaignCharacterCreationData{}, err
	}

	return CampaignCharacterCreationData{
		Progress: progress,
		Catalog:  catalog,
		Profile:  profile,
	}, nil
}

// campaignCharacterCreationProgress centralizes this web behavior in one helper seam.
func (s service) campaignCharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProgress, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignCharacterCreationProgress{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return CampaignCharacterCreationProgress{}, apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	return s.readGateway.CharacterCreationProgress(ctx, campaignID, characterID)
}

// applyCharacterCreationStep applies this package workflow transition.
func (s service) applyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *CampaignCharacterCreationStepInput) error {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	if step == nil {
		return apperrors.E(apperrors.KindInvalidInput, "character creation step is required")
	}
	if err := s.requirePolicyWithTarget(ctx, campaignID, policyMutateCharacter, characterID); err != nil {
		return err
	}
	return s.mutationGateway.ApplyCharacterCreationStep(ctx, campaignID, characterID, step)
}

// resetCharacterCreationWorkflow applies this package workflow transition.
func (s service) resetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	if err := s.requirePolicyWithTarget(ctx, campaignID, policyMutateCharacter, characterID); err != nil {
		return err
	}
	return s.mutationGateway.ResetCharacterCreationWorkflow(ctx, campaignID, characterID)
}
