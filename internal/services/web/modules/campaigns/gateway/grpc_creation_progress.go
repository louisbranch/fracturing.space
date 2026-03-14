package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// CharacterCreationProgress centralizes this web behavior in one helper seam.
func (g GRPCGateway) CharacterCreationProgress(ctx context.Context, campaignID string, characterID string) (campaignapp.CampaignCharacterCreationProgress, error) {
	if g.Read.Character == nil {
		return campaignapp.CampaignCharacterCreationProgress{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return campaignapp.CampaignCharacterCreationProgress{}, apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}

	resp, err := g.Read.Character.GetCharacterCreationProgress(ctx, &statev1.GetCharacterCreationProgressRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	if err != nil {
		return campaignapp.CampaignCharacterCreationProgress{}, err
	}
	if resp == nil || resp.GetProgress() == nil {
		return campaignapp.CampaignCharacterCreationProgress{Steps: []campaignapp.CampaignCharacterCreationStep{}, UnmetReasons: []string{}}, nil
	}

	progress := resp.GetProgress()
	steps := make([]campaignapp.CampaignCharacterCreationStep, 0, len(progress.GetSteps()))
	for _, step := range progress.GetSteps() {
		if step == nil {
			continue
		}
		steps = append(steps, campaignapp.CampaignCharacterCreationStep{
			Step:     step.GetStep(),
			Key:      strings.TrimSpace(step.GetKey()),
			Complete: step.GetComplete(),
		})
	}
	unmetReasons := make([]string, 0, len(progress.GetUnmetReasons()))
	for _, reason := range progress.GetUnmetReasons() {
		trimmedReason := strings.TrimSpace(reason)
		if trimmedReason == "" {
			continue
		}
		unmetReasons = append(unmetReasons, trimmedReason)
	}
	return campaignapp.CampaignCharacterCreationProgress{
		Steps:        steps,
		NextStep:     progress.GetNextStep(),
		Ready:        progress.GetReady(),
		UnmetReasons: unmetReasons,
	}, nil
}
