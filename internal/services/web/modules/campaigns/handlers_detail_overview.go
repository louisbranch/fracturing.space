package campaigns

import (
	"net/http"

	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
)

// handleOverview renders the default campaign detail overview section.
func (h handlers) handleOverview(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	view := page.overviewView(campaignID, h.authorization.RequireManageCampaign(ctx, campaignID) == nil)
	h.writeCampaignDetailPage(w, r, page, campaignID, campaignrender.OverviewFragment(view, page.loc))
}

// handleCampaignEdit handles this route in the module transport layer.
func (h handlers) handleCampaignEdit(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if err := h.authorization.RequireManageCampaign(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.campaignEditView(campaignID)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.CampaignEditFragment(view, page.loc),
		page.campaignEditBreadcrumbs(campaignID)...,
	)
}
