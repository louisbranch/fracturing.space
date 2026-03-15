package campaigns

import (
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// overviewView builds the overview detail view for one campaign.
func (p *campaignPageContext) overviewView(campaignID string, canEdit bool, summary campaignapp.CampaignAIBindingSummary) campaignrender.OverviewPageView {
	view := campaignrender.OverviewPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.CanEditCampaign = canEdit
	view.AIBindingStatus = string(summary.Status)
	view.CanManageAIBinding = summary.CanManage
	return view
}

// campaignEditView builds the overview-edit detail view for one campaign.
func (p *campaignPageContext) campaignEditView(campaignID string) campaignrender.CampaignEditPageView {
	view := campaignrender.CampaignEditPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.CanEditCampaign = true
	view.LocaleValue = campaignWorkspaceLocaleFormValue(view.Locale)
	return view
}

// campaignAIBindingView builds the dedicated AI-binding detail view for one campaign.
func (p *campaignPageContext) campaignAIBindingView(campaignID string, settings campaignapp.CampaignAIBindingSettings) campaignrender.CampaignAIBindingPageView {
	view := campaignrender.CampaignAIBindingPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.AIBindingSettings = mapAIBindingSettingsView(settings)
	return view
}

// campaignEditBreadcrumbs returns breadcrumbs for the overview edit page.
func (p *campaignPageContext) campaignEditBreadcrumbs(campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.campaign.menu.overview"), URL: routepath.AppCampaign(campaignID)},
		{Label: webtemplates.T(p.loc, "game.campaign.action_edit")},
	}
}

// campaignAIBindingBreadcrumbs returns breadcrumbs for the dedicated AI-binding page.
func (p *campaignPageContext) campaignAIBindingBreadcrumbs(campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.campaign.menu.overview"), URL: routepath.AppCampaign(campaignID)},
		{Label: webtemplates.T(p.loc, "game.campaign.ai_binding.title")},
	}
}

// mapAIBindingSettingsView converts campaign AI-binding app state into render state.
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
