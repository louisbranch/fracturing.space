package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// createCampaign executes package-scoped creation behavior for this flow.
func (s service) createCampaign(ctx context.Context, input CreateCampaignInput) (CreateCampaignResult, error) {
	if strings.TrimSpace(input.Name) == "" {
		return CreateCampaignResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_name_is_required", "campaign name is required")
	}
	created, err := s.mutationGateway.CreateCampaign(ctx, input)
	if err != nil {
		return CreateCampaignResult{}, err
	}
	if strings.TrimSpace(created.CampaignID) == "" {
		return CreateCampaignResult{}, apperrors.EK(apperrors.KindUnknown, "error.web.message.created_campaign_id_was_empty", "created campaign id was empty")
	}
	return created, nil
}

// updateCampaign applies this package workflow transition.
func (s service) updateCampaign(ctx context.Context, campaignID string, input UpdateCampaignInput) error {
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id is required")
	}
	if err := s.requireManageCampaign(ctx, campaignID); err != nil {
		return err
	}

	current, err := s.campaignWorkspace(ctx, campaignID)
	if err != nil {
		return err
	}

	patch, changed, err := buildCampaignUpdatePatch(input, current)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	return s.mutationGateway.UpdateCampaign(ctx, campaignID, patch)
}

// buildCampaignUpdatePatch maps validated campaign edit form input into a mutation patch.
func buildCampaignUpdatePatch(input UpdateCampaignInput, current CampaignWorkspace) (UpdateCampaignInput, bool, error) {
	patch := UpdateCampaignInput{}
	changed := false

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return UpdateCampaignInput{}, false, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_name_is_required", "campaign name is required")
		}
		if name != strings.TrimSpace(current.Name) {
			patch.Name = &name
			changed = true
		}
	}

	if input.ThemePrompt != nil {
		themePrompt := strings.TrimSpace(*input.ThemePrompt)
		if themePrompt != strings.TrimSpace(current.Theme) {
			patch.ThemePrompt = &themePrompt
			changed = true
		}
	}

	if input.Locale != nil {
		locale := campaignLocaleCanonical(*input.Locale)
		if locale == "" {
			return UpdateCampaignInput{}, false, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_locale_value_is_invalid", "campaign locale value is invalid")
		}
		if locale != campaignLocaleCanonical(current.Locale) {
			patch.Locale = &locale
			changed = true
		}
	}

	return patch, changed, nil
}
