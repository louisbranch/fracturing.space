package campaigns

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// campaignPageHandlerServices groups the shared workspace shell dependencies
// used by multiple campaign detail surfaces.
type campaignPageHandlerServices struct {
	workspace     campaignapp.CampaignWorkspaceService
	sessionReads  campaignapp.CampaignSessionReadService
	authorization campaignapp.CampaignAuthorizationService
}

// campaignDetailHandlers owns the shared workspace-shell route support used by
// detail, creation, session, and invite surfaces.
type campaignDetailHandlers struct {
	campaignRouteSupport
	pages campaignPageHandlerServices
}

// newCampaignPageHandlerServices keeps shared detail-page dependencies explicit
// at the campaign-page seam.
func newCampaignPageHandlerServices(config pageServiceConfig) campaignPageHandlerServices {
	return campaignPageHandlerServices{
		workspace:     campaignapp.NewWorkspaceService(config.Workspace),
		sessionReads:  campaignapp.NewSessionReadService(config.SessionRead),
		authorization: campaignapp.NewAuthorizationService(config.Authorization),
	}
}

// newCampaignDetailHandlers assembles shared detail-route support for the
// surfaces that render inside one campaign workspace shell.
func newCampaignDetailHandlers(support campaignRouteSupport, pages campaignPageHandlerServices) campaignDetailHandlers {
	return campaignDetailHandlers{
		campaignRouteSupport: support,
		pages:                pages,
	}
}

// missingCampaignPageHandlerServices reports which shared page-loading
// dependencies are absent before any detail route owner is constructed.
func missingCampaignPageHandlerServices(services campaignPageHandlerServices) []string {
	missing := []string{}
	if services.workspace == nil {
		missing = append(missing, "page-workspace")
	}
	if services.sessionReads == nil {
		missing = append(missing, "page-sessions")
	}
	if services.authorization == nil {
		missing = append(missing, "page-authorization")
	}
	return missing
}

// --- Campaign detail route handlers ---

// handleOverviewMethodNotAllowed preserves explicit Allow headers for the overview route.
func (h overviewHandlers) handleOverviewMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	httpx.MethodNotAllowed(http.MethodGet+", HEAD")(w, nil)
}

// loadCampaignPageOrWriteError loads common campaign detail page state and
// writes the transport error when loading fails.
func (h campaignDetailHandlers) loadCampaignPageOrWriteError(w http.ResponseWriter, r *http.Request, campaignID string) (context.Context, *campaignPageContext, bool) {
	ctx, page, err := h.loadCampaignPage(w, r, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return nil, nil, false
	}
	return ctx, page, true
}

// writeCampaignDetailPage renders one populated campaign detail view with the
// provided extra breadcrumbs.
func (h campaignDetailHandlers) writeCampaignDetailPage(
	w http.ResponseWriter,
	r *http.Request,
	page *campaignPageContext,
	campaignID string,
	body templ.Component,
	extra ...sharedtemplates.BreadcrumbItem,
) {
	h.writeCampaignDetailPageWithHeaderAction(w, r, page, campaignID, nil, body, extra...)
}

// writeCampaignDetailPageWithHeaderAction renders one populated campaign
// detail view with an optional header action and extra breadcrumbs.
func (h campaignDetailHandlers) writeCampaignDetailPageWithHeaderAction(
	w http.ResponseWriter,
	r *http.Request,
	page *campaignPageContext,
	campaignID string,
	action *webtemplates.AppMainHeaderAction,
	body templ.Component,
	extra ...sharedtemplates.BreadcrumbItem,
) {
	crumbs := campaignBreadcrumbs(campaignID, page.workspace.Name, page.loc, extra...)
	h.WritePage(w, r, page.title(campaignID), http.StatusOK,
		page.headerWithAction(campaignID, crumbs, action),
		page.layout(campaignID, r.URL.Path),
		body)
}
