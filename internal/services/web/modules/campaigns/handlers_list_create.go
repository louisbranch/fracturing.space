package campaigns

import (
	"fmt"
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/forminput"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// catalogHandlerServices groups campaign list and creation behavior.
type catalogHandlerServices struct {
	campaigns campaignapp.CampaignCatalogService
}

// catalogHandlers owns the campaign catalog and creation transport surface.
type catalogHandlers struct {
	campaignRouteSupport
	catalog catalogHandlerServices
	systems campaignSystemRegistry
}

// newCatalogHandlerServices keeps catalog transport dependencies owned by the
// list/create surface instead of the root constructor.
func newCatalogHandlerServices(config catalogServiceConfig) (catalogHandlerServices, error) {
	campaigns, err := campaignapp.NewCatalogService(config.Catalog)
	if err != nil {
		return catalogHandlerServices{}, fmt.Errorf("catalog: %w", err)
	}
	return catalogHandlerServices{campaigns: campaigns}, nil
}

// newCatalogHandlers assembles the catalog route-owner handler from support,
// services, and installed systems.
func newCatalogHandlers(support campaignRouteSupport, services catalogHandlerServices, systems campaignSystemRegistry) catalogHandlers {
	return catalogHandlers{
		campaignRouteSupport: support,
		catalog:              services,
		systems:              systems,
	}
}

// --- Headers ---

// campaignsListHeader builds the campaigns list header and primary creation action.
func campaignsListHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title: webtemplates.T(loc, "game.campaigns.title"),
		Action: &webtemplates.AppMainHeaderAction{
			Label: webtemplates.T(loc, "game.campaigns.start_new"),
			URL:   routepath.AppCampaignsNew,
		},
	}
}

// campaignStartHeader centralizes this web behavior in one helper seam.
func campaignStartHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title: webtemplates.T(loc, "game.campaigns.new.title"),
		Breadcrumbs: []sharedtemplates.BreadcrumbItem{
			{Label: webtemplates.T(loc, "game.campaigns.title"), URL: routepath.AppCampaigns},
			{Label: webtemplates.T(loc, "game.campaigns.new.title")},
		},
	}
}

// campaignCreateHeader centralizes this web behavior in one helper seam.
func campaignCreateHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title: webtemplates.T(loc, "game.create.title"),
		Breadcrumbs: []sharedtemplates.BreadcrumbItem{
			{Label: webtemplates.T(loc, "game.campaigns.title"), URL: routepath.AppCampaigns},
			{Label: webtemplates.T(loc, "game.create.title")},
		},
	}
}

// --- List and creation handlers ---

// handleIndex renders the campaign list page using the request-scoped service context.
func (h catalogHandlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	ctx, _ := h.RequestContextAndUserID(r)
	items, err := h.catalog.campaigns.ListCampaigns(ctx)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	if len(items) == 0 {
		flash.Write(w, r, flash.Notice{
			Kind: flash.KindInfo,
			Key:  "game.campaigns.empty",
		})
		httpx.WriteRedirect(w, r, routepath.AppCampaignsNew)
		return
	}
	h.WritePage(w, r, webtemplates.T(loc, "game.campaigns.title"), http.StatusOK, campaignsListHeader(loc), webtemplates.AppMainLayoutOptions{}, CampaignListFragment(mapCampaignListItems(items, h.now(), loc), loc))
}

// handleStartNewCampaign handles this route in the module transport layer.
func (h catalogHandlers) handleStartNewCampaign(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	h.WritePage(w, r,
		webtemplates.T(loc, "game.campaigns.new.title"), http.StatusOK,
		campaignStartHeader(loc),
		webtemplates.AppMainLayoutOptions{},
		CampaignStartFragment(loc),
	)
}

// handleCreateCampaign handles this route in the module transport layer.
func (h catalogHandlers) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	h.WritePage(w, r,
		webtemplates.T(loc, "game.create.title"), http.StatusOK,
		campaignCreateHeader(loc),
		webtemplates.AppMainLayoutOptions{},
		CampaignCreateFragment(CampaignCreateFormValues{}, h.systems.createOptions, h.systems.defaultCreateSystem(), loc),
	)
}

// handleCreateCampaignSubmit handles this route in the module transport layer.
func (h catalogHandlers) handleCreateCampaignSubmit(w http.ResponseWriter, r *http.Request) {
	if !forminput.ParseOrRedirectErrorNotice(w, r, "error.web.message.failed_to_parse_campaign_create_form", routepath.AppCampaignsCreate) {
		return
	}
	input, err := parseCreateCampaignInput(r.Form, h.systems)
	if err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_create_campaign", routepath.AppCampaignsCreate)
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	input.Locale = h.RequestLocaleTag(r)
	created, err := h.catalog.campaigns.CreateCampaign(ctx, input)
	if err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_create_campaign", routepath.AppCampaignsCreate)
		return
	}
	h.sync.CampaignCreated(ctx, userID, created.CampaignID)

	h.writeMutationSuccess(w, r, "web.campaigns.notice_campaign_created", routepath.AppCampaign(created.CampaignID))
}

// parseAppGmMode parses inbound values into package-safe forms.
func parseAppGmMode(value string) (campaignapp.GmMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "human":
		return campaignapp.GmModeHuman, true
	case "ai":
		return campaignapp.GmModeAI, true
	case "hybrid":
		return campaignapp.GmModeHybrid, true
	default:
		return campaignapp.GmModeUnspecified, false
	}
}
