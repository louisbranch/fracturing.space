package detail

import (
	"context"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// PageServiceConfig groups shared workspace-shell app config for detail
// surfaces.
type PageServiceConfig struct {
	Workspace     campaignapp.WorkspaceServiceConfig
	SessionRead   campaignapp.SessionReadServiceConfig
	Authorization campaignapp.AuthorizationGateway
}

// PageServices groups the shared workspace shell dependencies used by multiple
// campaign detail surfaces.
type PageServices struct {
	Workspace     campaignapp.CampaignWorkspaceService
	SessionReads  campaignapp.CampaignSessionReadService
	Authorization campaignapp.CampaignAuthorizationService
}

// NewPageServices keeps shared detail-page dependencies explicit at the
// campaign-page seam.
func NewPageServices(config PageServiceConfig) (PageServices, error) {
	workspace, err := campaignapp.NewWorkspaceService(config.Workspace)
	if err != nil {
		return PageServices{}, fmt.Errorf("page workspace: %w", err)
	}
	sessionReads, err := campaignapp.NewSessionReadService(config.SessionRead)
	if err != nil {
		return PageServices{}, fmt.Errorf("page sessions: %w", err)
	}
	authorization, err := campaignapp.NewAuthorizationService(config.Authorization)
	if err != nil {
		return PageServices{}, fmt.Errorf("page authorization: %w", err)
	}
	return PageServices{
		Workspace:     workspace,
		SessionReads:  sessionReads,
		Authorization: authorization,
	}, nil
}

// Handler owns the shared workspace-shell route support used by detail,
// creation, session, and invite surfaces.
type Handler struct {
	Support
	Pages PageServices
}

// NewHandler assembles shared detail-route support for the surfaces that render
// inside one campaign workspace shell.
func NewHandler(support Support, pages PageServices) Handler {
	return Handler{
		Support: support,
		Pages:   pages,
	}
}

// LoadCampaignPageOrWriteError loads common campaign detail page state and
// writes the transport error when loading fails.
func (h Handler) LoadCampaignPageOrWriteError(w http.ResponseWriter, r *http.Request, campaignID string) (context.Context, *PageContext, bool) {
	ctx, page, err := h.LoadCampaignPage(w, r, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return nil, nil, false
	}
	return ctx, page, true
}

// WriteCampaignDetailPage renders one populated campaign detail view with the
// provided extra breadcrumbs.
func (h Handler) WriteCampaignDetailPage(
	w http.ResponseWriter,
	r *http.Request,
	page *PageContext,
	campaignID string,
	body templ.Component,
	extra ...sharedtemplates.BreadcrumbItem,
) {
	h.WriteCampaignDetailPageWithHeaderAction(w, r, page, campaignID, nil, body, extra...)
}

// WriteCampaignDetailPageWithHeaderAction renders one populated campaign detail
// view with an optional header action and extra breadcrumbs.
func (h Handler) WriteCampaignDetailPageWithHeaderAction(
	w http.ResponseWriter,
	r *http.Request,
	page *PageContext,
	campaignID string,
	action *webtemplates.AppMainHeaderAction,
	body templ.Component,
	extra ...sharedtemplates.BreadcrumbItem,
) {
	crumbs := CampaignBreadcrumbs(campaignID, page.Workspace.Name, page.Loc, extra...)
	h.WritePage(w, r, page.Title(campaignID), http.StatusOK,
		page.HeaderWithAction(campaignID, crumbs, action),
		page.Layout(campaignID, r.URL.Path),
		body)
}
