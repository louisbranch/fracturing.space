package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

// CampaignCharacterCreationProgress centralizes this web behavior in one helper seam.
func (s creationPageService) CampaignCharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProgress, error) {
	return s.campaignCharacterCreationProgress(ctx, campaignID, characterID)
}

// CampaignCharacterCreationCatalog centralizes character-creation catalog reads
// in one helper seam.
func (s creationPageService) CampaignCharacterCreationCatalog(ctx context.Context, locale language.Tag) (CampaignCharacterCreationCatalog, error) {
	return s.campaignCharacterCreationCatalog(ctx, locale)
}

// CampaignCharacterCreationProfile centralizes character-creation profile reads
// in one helper seam.
func (s creationPageService) CampaignCharacterCreationProfile(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProfile, error) {
	return s.campaignCharacterCreationProfile(ctx, campaignID, characterID)
}

// CampaignCharacterCreationProgress centralizes this web behavior in one helper seam.
func (s creationMutationService) CampaignCharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProgress, error) {
	return s.read.CharacterCreationProgress(ctx, campaignID, characterID)
}

// ApplyCharacterCreationStep applies this package workflow transition.
func (s creationMutationService) ApplyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *CampaignCharacterCreationStepInput) error {
	return s.applyCharacterCreationStep(ctx, campaignID, characterID, step)
}

// ResetCharacterCreationWorkflow applies this package workflow transition.
func (s creationMutationService) ResetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	return s.resetCharacterCreationWorkflow(ctx, campaignID, characterID)
}

// campaignCharacterCreationProgress centralizes this web behavior in one helper seam.
func (s creationPageService) campaignCharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProgress, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignCharacterCreationProgress{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return CampaignCharacterCreationProgress{}, apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	return s.read.CharacterCreationProgress(ctx, campaignID, characterID)
}

// campaignCharacterCreationCatalog centralizes catalog reads used by
// character-creation pages in one helper seam.
func (s creationPageService) campaignCharacterCreationCatalog(ctx context.Context, locale language.Tag) (CampaignCharacterCreationCatalog, error) {
	return s.read.CharacterCreationCatalog(ctx, locale)
}

// campaignCharacterCreationProfile centralizes profile reads used by
// character-creation pages in one helper seam.
func (s creationPageService) campaignCharacterCreationProfile(ctx context.Context, campaignID string, characterID string) (CampaignCharacterCreationProfile, error) {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return CampaignCharacterCreationProfile{}, apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return CampaignCharacterCreationProfile{}, apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	return s.read.CharacterCreationProfile(ctx, campaignID, characterID)
}

// applyCharacterCreationStep applies this package workflow transition.
func (s creationMutationService) applyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *CampaignCharacterCreationStepInput) error {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	if step == nil {
		return apperrors.E(apperrors.KindInvalidInput, "character creation step is required")
	}
	if err := s.auth.requirePolicyWithTarget(ctx, campaignID, policyMutateCharacter, characterID); err != nil {
		return err
	}
	return s.mutation.ApplyCharacterCreationStep(ctx, campaignID, characterID, step)
}

// resetCharacterCreationWorkflow applies this package workflow transition.
func (s creationMutationService) resetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "character id is required")
	}
	if err := s.auth.requirePolicyWithTarget(ctx, campaignID, policyMutateCharacter, characterID); err != nil {
		return err
	}
	return s.mutation.ResetCharacterCreationWorkflow(ctx, campaignID, characterID)
}
