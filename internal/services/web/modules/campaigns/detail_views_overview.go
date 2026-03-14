package campaigns

import (
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// overviewView builds the overview detail view for one campaign.
func (p *campaignPageContext) overviewView(campaignID string, canEdit bool) campaignrender.OverviewPageView {
	view := campaignrender.OverviewPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.CanEditCampaign = canEdit
	return view
}

// campaignEditView builds the overview-edit detail view for one campaign.
func (p *campaignPageContext) campaignEditView(campaignID string) campaignrender.CampaignEditPageView {
	view := campaignrender.CampaignEditPageView{CampaignDetailBaseView: p.baseDetailView(campaignID)}
	view.CanEditCampaign = true
	view.LocaleValue = campaignWorkspaceLocaleFormValue(view.Locale)
	return view
}

// campaignEditBreadcrumbs returns breadcrumbs for the overview edit page.
func (p *campaignPageContext) campaignEditBreadcrumbs(campaignID string) []sharedtemplates.BreadcrumbItem {
	return []sharedtemplates.BreadcrumbItem{
		{Label: webtemplates.T(p.loc, "game.campaign.menu.overview"), URL: routepath.AppCampaign(campaignID)},
		{Label: webtemplates.T(p.loc, "game.campaign.action_edit")},
	}
}
