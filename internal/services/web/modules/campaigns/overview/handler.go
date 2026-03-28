package overview

import (
	"fmt"
	"net/http"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// ServiceConfig groups overview, AI binding, and campaign settings app config.
type ServiceConfig struct {
	AutomationRead     campaignapp.AutomationReadServiceConfig
	AutomationMutation campaignapp.AutomationMutationServiceConfig
	Configuration      campaignapp.ConfigurationServiceConfig
	Authorization      campaignapp.AuthorizationGateway
}

// HandlerServices groups campaign overview, configuration, and AI-binding
// behavior.
type HandlerServices struct {
	automationReads  campaignapp.CampaignAutomationReadService
	automationMutate campaignapp.CampaignAutomationMutationService
	configuration    campaignapp.CampaignConfigurationService
}

// NewHandlerServices keeps overview transport dependencies owned by the
// overview surface instead of the campaigns root constructor.
func NewHandlerServices(config ServiceConfig) (HandlerServices, error) {
	automationReads, err := campaignapp.NewAutomationReadService(config.AutomationRead, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("overview automation-reads: %w", err)
	}
	automationMutate, err := campaignapp.NewAutomationMutationService(config.AutomationMutation, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("overview automation-mutation: %w", err)
	}
	configuration, err := campaignapp.NewConfigurationService(config.Configuration, config.Authorization)
	if err != nil {
		return HandlerServices{}, fmt.Errorf("overview configuration: %w", err)
	}
	return HandlerServices{
		automationReads:  automationReads,
		automationMutate: automationMutate,
		configuration:    configuration,
	}, nil
}

// Handler owns overview, configuration, and AI-binding routes.
type Handler struct {
	campaigndetail.Handler
	overview HandlerServices
}

// NewHandler assembles the overview route-owner handler.
func NewHandler(detail campaigndetail.Handler, services HandlerServices) Handler {
	return Handler{
		Handler:  detail,
		overview: services,
	}
}

// HandleOverviewMethodNotAllowed preserves explicit Allow headers for the
// overview route.
func (h Handler) HandleOverviewMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	httpx.MethodNotAllowed(http.MethodGet+", HEAD")(w, nil)
}

// HandleOverview renders the default campaign detail overview section.
func (h Handler) HandleOverview(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	summary, err := h.overview.automationReads.CampaignAIBindingSummary(ctx, campaignID, page.Workspace.AIAgentID, page.Workspace.GMMode)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := overviewView(page, campaignID, h.Pages.Authorization.RequireManageCampaign(ctx, campaignID) == nil, summary)
	h.WriteCampaignDetailPage(w, r, page, campaignID, campaignrender.OverviewFragment(view, page.Loc))
}

// HandleCampaignEdit renders the overview edit page.
func (h Handler) HandleCampaignEdit(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	if err := h.Pages.Authorization.RequireManageCampaign(ctx, campaignID); err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := campaignEditView(page, campaignID)
	h.WriteCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.CampaignEditFragment(view, page.Loc),
		campaignEditBreadcrumbs(page, campaignID)...,
	)
}

// HandleCampaignAIBindingPage renders the dedicated campaign AI-binding page.
func (h Handler) HandleCampaignAIBindingPage(w http.ResponseWriter, r *http.Request, campaignID string) {
	ctx, page, ok := h.LoadCampaignPageOrWriteError(w, r, campaignID)
	if !ok {
		return
	}
	settings, err := h.overview.automationReads.CampaignAIBindingSettings(ctx, campaignID, page.Workspace.AIAgentID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := campaignAIBindingView(page, campaignID, settings)
	h.WriteCampaignDetailPage(
		w,
		r,
		page,
		campaignID,
		campaignrender.CampaignAIBindingFragment(view, page.Loc),
		campaignAIBindingBreadcrumbs(page, campaignID)...,
	)
}

// HandleCampaignAIBinding updates the AI binding for one campaign.
func (h Handler) HandleCampaignAIBinding(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_campaign_ai_binding_form", routepath.AppCampaignAIBinding(campaignID)) {
		return
	}
	input := parseUpdateCampaignAIBindingInput(r.Form)
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.overview.automationMutate.UpdateCampaignAIBinding(ctx, campaignID, input); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_update_ai_binding", routepath.AppCampaignAIBinding(campaignID))
		return
	}
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_ai_binding_saved", routepath.AppCampaign(campaignID))
}

// HandleCampaignUpdate updates the campaign metadata from the edit page.
func (h Handler) HandleCampaignUpdate(w http.ResponseWriter, r *http.Request, campaignID string) {
	if !httpx.ParseFormOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_campaign_update_form", routepath.AppCampaign(campaignID)) {
		return
	}
	ctx, _ := h.RequestContextAndUserID(r)
	if err := h.overview.configuration.UpdateCampaign(ctx, campaignID, parseUpdateCampaignInput(r.Form)); err != nil {
		h.WriteMutationError(w, r, err, "error.web.message.failed_to_update_campaign", routepath.AppCampaign(campaignID))
		return
	}
	h.WriteMutationSuccess(w, r, "web.campaigns.notice_campaign_updated", routepath.AppCampaign(campaignID))
}
