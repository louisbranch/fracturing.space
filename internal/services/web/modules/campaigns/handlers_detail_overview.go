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
	summary, err := h.automationReads.CampaignAIBindingSummary(ctx, campaignID, page.workspace.AIAgentID, page.workspace.GMMode)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.overviewView(campaignID, h.authorization.RequireManageCampaign(ctx, campaignID) == nil, summary)
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

// handleCampaignAIBindingPage renders the dedicated campaign AI-binding page.
func (h handlers) handleCampaignAIBindingPage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	settings, err := h.automationReads.CampaignAIBindingSettings(ctx, campaignID, page.workspace.AIAgentID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.campaignAIBindingView(campaignID, settings)
	h.writeCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.CampaignAIBindingFragment(view, page.loc),
		page.campaignAIBindingBreadcrumbs(campaignID)...,
	)
}
