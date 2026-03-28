package render

import (
	"strings"

	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// campaignOverviewTheme preserves the explicit empty-theme fallback copy.
func campaignOverviewTheme(loc webtemplates.Localizer, theme string) string {
	if strings.TrimSpace(theme) == "" {
		return webtemplates.T(loc, "game.campaign.overview.theme_empty")
	}
	return strings.TrimSpace(theme)
}

// campaignOverviewSystem adapts the detail view to the shared system-label helper.
func campaignOverviewSystem(loc webtemplates.Localizer, system string) string {
	return campaignSystemLabel(loc, system)
}

// campaignOverviewGMMode adapts the detail view to the shared GM-mode label helper.
func campaignOverviewGMMode(loc webtemplates.Localizer, gmMode string) string {
	return campaignGMModeLabel(loc, gmMode)
}

// campaignOverviewStatus keeps campaign status labels consistent across detail screens.
func campaignOverviewStatus(loc webtemplates.Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return webtemplates.T(loc, "game.campaign.system_unspecified")
	case "draft":
		return webtemplates.T(loc, "game.campaign.overview.value_draft")
	case "active":
		return webtemplates.T(loc, "game.campaign.overview.value_active")
	case "completed":
		return webtemplates.T(loc, "game.campaign.overview.value_completed")
	case "archived":
		return webtemplates.T(loc, "game.campaign.overview.value_archived")
	default:
		return raw
	}
}

// campaignOverviewLocale maps persisted locale labels to the user-facing copy.
func campaignOverviewLocale(loc webtemplates.Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return webtemplates.T(loc, "game.campaign.system_unspecified")
	case "en_us":
		return webtemplates.T(loc, "game.campaign.overview.value_locale_en_us")
	case "pt_br":
		return webtemplates.T(loc, "game.campaign.overview.value_locale_pt_br")
	default:
		return raw
	}
}

// campaignOverviewIntent maps campaign intent values to localized detail-page text.
func campaignOverviewIntent(loc webtemplates.Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return webtemplates.T(loc, "game.campaign.system_unspecified")
	case "standard":
		return webtemplates.T(loc, "game.campaign.overview.value_standard")
	case "starter":
		return webtemplates.T(loc, "game.campaign.overview.value_starter")
	case "sandbox":
		return webtemplates.T(loc, "game.campaign.overview.value_sandbox")
	default:
		return raw
	}
}

// campaignOverviewAccessPolicy maps access-policy values to localized detail-page text.
func campaignOverviewAccessPolicy(loc webtemplates.Localizer, value string) string {
	raw := strings.TrimSpace(value)
	switch strings.ToLower(raw) {
	case "", "unspecified":
		return webtemplates.T(loc, "game.campaign.system_unspecified")
	case "private":
		return webtemplates.T(loc, "game.campaign.overview.value_private")
	case "restricted":
		return webtemplates.T(loc, "game.campaign.overview.value_restricted")
	case "public":
		return webtemplates.T(loc, "game.campaign.overview.value_public")
	default:
		return raw
	}
}

// campaignOverviewAIBindingStatus maps campaign AI-binding status values to localized copy.
func campaignOverviewAIBindingStatus(loc webtemplates.Localizer, value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "configured":
		return webtemplates.T(loc, "game.campaign.ai_binding.status_configured")
	case "pending":
		return webtemplates.T(loc, "game.campaign.ai_binding.status_pending")
	default:
		return webtemplates.T(loc, "game.campaign.ai_binding.status_not_required")
	}
}

// campaignAIBindingCurrentUnbound drives the unbound-state copy for campaign AI-binding forms.
func campaignAIBindingCurrentUnbound(settings AIBindingSettingsView) bool {
	return strings.TrimSpace(settings.CurrentID) == ""
}
