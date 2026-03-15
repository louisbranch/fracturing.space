package app

import (
	"context"

	"golang.org/x/text/language"
)

// CharacterCreationReadGateway loads character-creation reads for the web workflow surface.
type CharacterCreationReadGateway interface {
	CharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error)
	CharacterCreationCatalog(context.Context, language.Tag) (CampaignCharacterCreationCatalog, error)
	CharacterCreationProfile(context.Context, string, string) (CampaignCharacterCreationProfile, error)
}

// CharacterCreationMutationGateway applies character-creation workflow mutations for the web service.
type CharacterCreationMutationGateway interface {
	ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error
	ResetCharacterCreationWorkflow(context.Context, string, string) error
}
