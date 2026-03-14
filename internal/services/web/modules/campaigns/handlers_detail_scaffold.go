package campaigns

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
)

// --- Campaign detail route handlers ---

// handleOverviewMethodNotAllowed preserves explicit Allow headers for the overview route.
func (h handlers) handleOverviewMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	httpx.MethodNotAllowed(http.MethodGet+", HEAD")(w, nil)
}

// loadCampaignPageOrWriteError loads common campaign detail page state and
// writes the transport error when loading fails.
func (h handlers) loadCampaignPageOrWriteError(w http.ResponseWriter, r *http.Request, campaignID string) (context.Context, *campaignPageContext, bool) {
	ctx, page, err := h.loadCampaignPage(w, r, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return nil, nil, false
	}
	return ctx, page, true
}

// writeCampaignDetailPage renders one populated campaign detail view with the
// provided extra breadcrumbs.
func (h handlers) writeCampaignDetailPage(
	w http.ResponseWriter,
	r *http.Request,
	page *campaignPageContext,
	campaignID string,
	body templ.Component,
	extra ...sharedtemplates.BreadcrumbItem,
) {
	crumbs := campaignBreadcrumbs(campaignID, page.workspace.Name, page.loc, extra...)
	h.WritePage(w, r, page.title(campaignID), http.StatusOK,
		page.header(campaignID, crumbs),
		page.layout(campaignID, r.URL.Path),
		body)
}
