package app

import (
	"context"

	"golang.org/x/text/language"
)

// CampaignCharacterCreationPageService exposes character-creation page reads.
type CampaignCharacterCreationPageService interface {
	CampaignCharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	CampaignCharacterCreationCatalog(context.Context, language.Tag) (CampaignCharacterCreationCatalog, error)
	CampaignCharacterCreationProfile(context.Context, string, string) (CampaignCharacterCreationProfile, error)
}

// CampaignCharacterCreationMutationService exposes character-creation workflow progress reads and mutations.
type CampaignCharacterCreationMutationService interface {
	CampaignCharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error
	ResetCharacterCreationWorkflow(context.Context, string, string) error
}
