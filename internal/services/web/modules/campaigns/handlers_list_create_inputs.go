package campaigns

import (
	"net/url"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// parseCreateCampaignInput maps and validates create-campaign form values.
func parseCreateCampaignInput(form url.Values) (CreateCampaignInput, error) {
	systemValue := strings.TrimSpace(form.Get("system"))
	if systemValue == "" {
		systemValue = "daggerheart"
	}
	system, ok := parseAppGameSystem(systemValue)
	if !ok {
		return CreateCampaignInput{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_system_is_invalid", "campaign system is invalid")
	}

	gmModeValue := strings.TrimSpace(form.Get("gm_mode"))
	if gmModeValue == "" {
		gmModeValue = "ai"
	}
	gmMode, ok := parseAppGmMode(gmModeValue)
	if !ok {
		return CreateCampaignInput{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_gm_mode_is_invalid", "campaign gm mode is invalid")
	}

	return CreateCampaignInput{
		Name:        strings.TrimSpace(form.Get("name")),
		System:      system,
		GMMode:      gmMode,
		ThemePrompt: strings.TrimSpace(form.Get("theme_prompt")),
	}, nil
}
