package render

import "github.com/a-h/templ"

// OverviewPageView carries overview-page state only.
type OverviewPageView struct {
	CampaignDetailBaseView
}

// CampaignEditPageView carries overview-edit page state only.
type CampaignEditPageView struct {
	CampaignDetailBaseView
}

// OverviewFragment renders the campaign overview page.
func OverviewFragment(view OverviewPageView, loc Localizer) templ.Component {
	return overviewFragment(view, loc)
}

// CampaignEditFragment renders the campaign overview-edit page.
func CampaignEditFragment(view CampaignEditPageView, loc Localizer) templ.Component {
	return campaignEditFragment(view, loc)
}
