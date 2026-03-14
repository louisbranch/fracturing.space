package campaigns

import (
	"net/http"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handleOverview renders the default campaign detail overview section.
func (h handlers) handleOverview(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	view := page.detailView(campaignID, markerOverview)
	if err := h.service.RequireManageCampaign(ctx, campaignID); err == nil {
		view.CanEditCampaign = true
	}
	h.writeCampaignDetailPage(w, r, page, campaignID, view)
}

// handleCampaignEdit handles this route in the module transport layer.
func (h handlers) handleCampaignEdit(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	view := page.detailView(campaignID, markerCampaignEdit)
	if err := h.service.RequireManageCampaign(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	view.CanEditCampaign = true
	view.LocaleValue = campaignWorkspaceLocaleFormValue(view.Locale)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		view,
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.campaign.menu.overview"), URL: routepath.AppCampaign(campaignID)},
		sharedtemplates.BreadcrumbItem{Label: webtemplates.T(page.loc, "game.campaign.action_edit")},
	)
}
