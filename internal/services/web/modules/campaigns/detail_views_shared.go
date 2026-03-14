package campaigns

import campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"

// baseDetailView maps shared campaign workspace state into the base detail
// render model consumed by section-owned detail view builders.
func (p *campaignPageContext) baseDetailView(campaignID string) campaignrender.CampaignDetailBaseView {
	return campaignrender.CampaignDetailBaseView{
		CampaignID:       campaignID,
		Name:             p.workspace.Name,
		Theme:            p.workspace.Theme,
		System:           p.workspace.System,
		GMMode:           p.workspace.GMMode,
		Status:           p.workspace.Status,
		Locale:           p.workspace.Locale,
		LocaleValue:      campaignWorkspaceLocaleFormValue(p.workspace.Locale),
		Intent:           p.workspace.Intent,
		AccessPolicy:     p.workspace.AccessPolicy,
		ActionsLocked:    p.outOfGameActionsLocked(),
		CanManageInvites: p.canManageInvites,
	}
}
