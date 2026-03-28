package overview

import (
	"net/url"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// overviewView builds the overview page view from campaign status inputs.
func overviewView(page *campaigndetail.PageContext, campaignID string, canEdit bool, summary campaignapp.CampaignAIBindingSummary) campaignrender.OverviewPageView {
	view := campaignrender.OverviewPageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.CanEditCampaign = canEdit
	view.AIBindingStatus = string(summary.Status)
	view.CanManageAIBinding = summary.CanManage
	return view
}

// campaignEditView builds the campaign-edit page view from workspace data.
func campaignEditView(page *campaigndetail.PageContext, campaignID string) campaignrender.CampaignEditPageView {
	view := campaignrender.CampaignEditPageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.CanEditCampaign = true
	view.LocaleValue = campaigndetail.CampaignWorkspaceLocaleFormValue(view.Locale)
	return view
}

// campaignAIBindingView builds the AI-binding page view from current settings.
func campaignAIBindingView(page *campaigndetail.PageContext, campaignID string, settings campaignapp.CampaignAIBindingSettings) campaignrender.CampaignAIBindingPageView {
	view := campaignrender.CampaignAIBindingPageView{CampaignDetailBaseView: page.BaseDetailView(campaignID)}
	view.AIBindingSettings = mapAIBindingSettingsView(settings)
	return view
}

// campaignEditBreadcrumbs returns breadcrumbs for the campaign-edit page.
func campaignEditBreadcrumbs(page *campaigndetail.PageContext, campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.campaign.menu.overview"), URL: routepath.AppCampaign(campaignID)},
		{Label: webtemplates.T(page.Loc, "game.campaign.action_edit")},
	}
}

// campaignAIBindingBreadcrumbs returns breadcrumbs for the AI-binding page.
func campaignAIBindingBreadcrumbs(page *campaigndetail.PageContext, campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(page.Loc, "game.campaign.menu.overview"), URL: routepath.AppCampaign(campaignID)},
		{Label: webtemplates.T(page.Loc, "game.campaign.ai_binding.title")},
	}
}

// mapAIBindingSettingsView projects app-layer AI settings into render state.
func mapAIBindingSettingsView(settings campaignapp.CampaignAIBindingSettings) campaignrender.AIBindingSettingsView {
	options := make([]campaignrender.AIAgentOptionView, 0, len(settings.Options))
	for _, option := range settings.Options {
		options = append(options, campaignrender.AIAgentOptionView{
			ID:       option.ID,
			Name:     option.Label,
			Enabled:  option.Enabled,
			Selected: option.Selected,
		})
	}
	return campaignrender.AIBindingSettingsView{
		Unavailable: settings.Unavailable,
		CurrentID:   settings.CurrentID,
		Options:     options,
	}
}

// parseUpdateCampaignAIBindingInput normalizes the AI binding update form.
func parseUpdateCampaignAIBindingInput(form url.Values) campaignapp.UpdateCampaignAIBindingInput {
	return campaignapp.UpdateCampaignAIBindingInput{
		AIAgentID: strings.TrimSpace(form.Get("ai_agent_id")),
	}
}

// parseUpdateCampaignInput normalizes the campaign edit form into pointer fields.
func parseUpdateCampaignInput(form url.Values) campaignapp.UpdateCampaignInput {
	name := strings.TrimSpace(form.Get("name"))
	themePrompt := strings.TrimSpace(form.Get("theme_prompt"))
	locale := strings.TrimSpace(form.Get("locale"))
	return campaignapp.UpdateCampaignInput{
		Name:        &name,
		ThemePrompt: &themePrompt,
		Locale:      &locale,
	}
}
