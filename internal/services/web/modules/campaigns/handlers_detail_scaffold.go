package campaigns

import (
	"context"
	"net/http"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// --- Campaign detail route handlers ---

// handleOverviewMethodNotAllowed preserves explicit Allow headers for the overview route.
func (h handlers) handleOverviewMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	httpx.MethodNotAllowed(http.MethodGet+", HEAD")(w, nil)
}

// --- Campaign detail scaffold ---

// campaignDetailSpec describes one campaign sub-page. The scaffold loads the
// campaign workspace, calls loadData to populate view-specific fields, builds
// breadcrumbs, and renders the detail fragment.
type campaignDetailSpec struct {
	marker   string
	extra    func(loc webtemplates.Localizer, view webtemplates.CampaignDetailView) []sharedtemplates.BreadcrumbItem
	loadData func(ctx context.Context, campaignID string, page *campaignPageContext, view *webtemplates.CampaignDetailView) error
}

// renderCampaignDetail centralizes this web behavior in one helper seam.
func (h handlers) renderCampaignDetail(w http.ResponseWriter, r *http.Request, campaignID string, spec campaignDetailSpec) {
	ctx, page, err := h.loadCampaignPage(w, r, campaignID)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	view := page.detailView(campaignID, spec.marker)
	if spec.loadData != nil {
		if err := spec.loadData(ctx, campaignID, page, &view); err != nil {
			h.WriteError(w, r, err)
			return
		}
	}
	var crumbs []sharedtemplates.BreadcrumbItem
	if spec.extra != nil {
		crumbs = campaignBreadcrumbs(campaignID, page.workspace.Name, page.loc, spec.extra(page.loc, view)...)
	} else {
		crumbs = campaignBreadcrumbs(campaignID, page.workspace.Name, page.loc)
	}
	h.WritePage(w, r, page.title(campaignID), http.StatusOK,
		page.header(campaignID, crumbs),
		page.layout(campaignID, r.URL.Path),
		webtemplates.CampaignDetailFragment(view, page.loc))
}
