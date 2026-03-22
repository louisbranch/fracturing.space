package campaigns

import (
	"fmt"
	"net/http"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
)

// overviewHandlerServices groups campaign overview, configuration, and AI
// binding behavior.
type overviewHandlerServices struct {
	automationReads  campaignapp.CampaignAutomationReadService
	automationMutate campaignapp.CampaignAutomationMutationService
	configuration    campaignapp.CampaignConfigurationService
}

// overviewHandlers owns overview/configuration/AI-binding routes.
type overviewHandlers struct {
	campaignDetailHandlers
	overview overviewHandlerServices
}

// newOverviewHandlerServices keeps overview transport dependencies owned by the
// overview surface instead of the root constructor.
func newOverviewHandlerServices(config overviewServiceConfig) (overviewHandlerServices, error) {
	automationReads, err := campaignapp.NewAutomationReadService(config.AutomationRead, config.Authorization)
	if err != nil {
		return overviewHandlerServices{}, fmt.Errorf("overview automation-reads: %w", err)
	}
	automationMutate, err := campaignapp.NewAutomationMutationService(config.AutomationMutation, config.Authorization)
	if err != nil {
		return overviewHandlerServices{}, fmt.Errorf("overview automation-mutation: %w", err)
	}
	configuration, err := campaignapp.NewConfigurationService(config.Configuration, config.Authorization)
	if err != nil {
		return overviewHandlerServices{}, fmt.Errorf("overview configuration: %w", err)
	}
	return overviewHandlerServices{
		automationReads:  automationReads,
		automationMutate: automationMutate,
		configuration:    configuration,
	}, nil
}

// newOverviewHandlers assembles the overview route-owner handler.
func newOverviewHandlers(detail campaignDetailHandlers, services overviewHandlerServices) overviewHandlers {
	return overviewHandlers{
		campaignDetailHandlers: detail,
		overview:               services,
	}
}

// handleOverview renders the default campaign detail overview section.
func (h overviewHandlers) handleOverview(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	summary, err := h.overview.automationReads.CampaignAIBindingSummary(ctx, campaignID, page.workspace.AIAgentID, page.workspace.GMMode)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.overviewView(campaignID, h.pages.authorization.RequireManageCampaign(ctx, campaignID) == nil, summary)
	h.writeCampaignDetailPage(w, r, page, campaignID, campaignrender.OverviewFragment(view, page.loc))
}

// handleCampaignEdit handles this route in the module transport layer.
func (h overviewHandlers) handleCampaignEdit(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if err := h.pages.authorization.RequireManageCampaign(ctx, campaignID); err != nil {
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
func (h overviewHandlers) handleCampaignAIBindingPage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.loadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	settings, err := h.overview.automationReads.CampaignAIBindingSettings(ctx, campaignID, page.workspace.AIAgentID)
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
