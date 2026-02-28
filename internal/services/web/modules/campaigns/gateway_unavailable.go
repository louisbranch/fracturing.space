package campaigns

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

type unavailableGateway struct{}

func (unavailableGateway) ListCampaigns(context.Context) ([]CampaignSummary, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignName(context.Context, string) (string, error) {
	return "", apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignWorkspace(context.Context, string) (CampaignWorkspace, error) {
	return CampaignWorkspace{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignParticipants(context.Context, string) ([]CampaignParticipant, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignCharacters(context.Context, string) ([]CampaignCharacter, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignSessions(context.Context, string) ([]CampaignSession, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CampaignInvites(context.Context, string) ([]CampaignInvite, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CharacterCreationProgress(context.Context, string, string) (CampaignCharacterCreationProgress, error) {
	return CampaignCharacterCreationProgress{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CharacterCreationCatalog(context.Context, language.Tag) (CampaignCharacterCreationCatalog, error) {
	return CampaignCharacterCreationCatalog{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CharacterCreationProfile(context.Context, string, string) (CampaignCharacterCreationProfile, error) {
	return CampaignCharacterCreationProfile{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CreateCampaign(context.Context, CreateCampaignInput) (CreateCampaignResult, error) {
	return CreateCampaignResult{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) StartSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) EndSession(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) UpdateParticipants(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CreateCharacter(context.Context, string, CreateCharacterInput) (CreateCharacterResult, error) {
	return CreateCharacterResult{}, apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) UpdateCharacter(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) ControlCharacter(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) CreateInvite(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) RevokeInvite(context.Context, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) ApplyCharacterCreationStep(context.Context, string, string, *CampaignCharacterCreationStepInput) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}

func (unavailableGateway) ResetCharacterCreationWorkflow(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "campaigns service is not configured")
}
