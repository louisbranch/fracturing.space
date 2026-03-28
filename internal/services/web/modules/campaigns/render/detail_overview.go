package render

import (
	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// OverviewPageView carries overview-page state only.
type OverviewPageView struct {
	CampaignDetailBaseView
	AIBindingStatus    string
	CanManageAIBinding bool
}

// CampaignEditPageView carries overview-edit page state only.
type CampaignEditPageView struct {
	CampaignDetailBaseView
}

// CampaignAIBindingPageView carries dedicated campaign AI-binding page state.
type CampaignAIBindingPageView struct {
	CampaignDetailBaseView
	AIBindingSettings AIBindingSettingsView
}

// AIBindingSettingsView keeps campaign AI-binding form state local to
// overview-owned rendering.
type AIBindingSettingsView struct {
	Unavailable bool
	CurrentID   string
	Options     []AIAgentOptionView
}

// OverviewFragment renders the campaign overview page.
func OverviewFragment(view OverviewPageView, loc webtemplates.Localizer) templ.Component {
	return overviewFragment(view, loc)
}

// CampaignEditFragment renders the campaign overview-edit page.
func CampaignEditFragment(view CampaignEditPageView, loc webtemplates.Localizer) templ.Component {
	return campaignEditFragment(view, loc)
}

// CampaignAIBindingFragment renders the dedicated campaign AI-binding page.
func CampaignAIBindingFragment(view CampaignAIBindingPageView, loc webtemplates.Localizer) templ.Component {
	return campaignAIBindingFragment(view, loc)
}
