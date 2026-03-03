package campaigns

import (
	"context"
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// --- Campaign detail route handlers ---

// handleOverviewMethodNotAllowed preserves explicit Allow headers for the overview route.
func (h handlers) handleOverviewMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	httpx.MethodNotAllowed(http.MethodGet+", HEAD")(w, nil)
}

// handleCharacterDetailRoute handles this route in the module transport layer.
func (h handlers) handleCharacterDetailRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	characterID, ok := h.routeCharacterID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	h.handleCharacterDetail(w, r, campaignID, characterID)
}

// handleParticipantEditRoute handles this route in the module transport layer.
func (h handlers) handleParticipantEditRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	participantID, ok := h.routeParticipantID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	h.handleParticipantEdit(w, r, campaignID, participantID)
}

// handleCampaignEditRoute handles this route in the module transport layer.
func (h handlers) handleCampaignEditRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	h.handleCampaignEdit(w, r, campaignID)
}

// handleSessionDetailRoute handles this route in the module transport layer.
func (h handlers) handleSessionDetailRoute(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.routeCampaignID(r)
	if !ok {
		h.WriteNotFound(w, r)
		return
	}
	sessionID := strings.TrimSpace(r.PathValue("sessionID"))
	if sessionID == "" {
		h.WriteNotFound(w, r)
		return
	}
	h.handleSessionDetail(w, r, campaignID, sessionID)
}

// --- Campaign detail scaffold ---

// campaignDetailSpec describes one campaign sub-page. The scaffold loads the
// campaign workspace, calls loadData to populate view-specific fields, builds
// breadcrumbs, and renders the detail fragment.
type campaignDetailSpec struct {
	marker   string
	extra    func(loc webtemplates.Localizer) []sharedtemplates.BreadcrumbItem
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
		crumbs = campaignBreadcrumbs(campaignID, page.workspace.Name, page.loc, spec.extra(page.loc)...)
	} else {
		crumbs = campaignBreadcrumbs(campaignID, page.workspace.Name, page.loc)
	}
	h.WritePage(w, r, page.title(campaignID), http.StatusOK,
		page.header(campaignID, crumbs),
		page.layout(campaignID, r.URL.Path),
		webtemplates.CampaignDetailFragment(view, page.loc))
}
