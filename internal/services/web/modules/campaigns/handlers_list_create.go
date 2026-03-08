package campaigns

import (
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

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
func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	ctx, _ := h.RequestContextAndUserID(r)
	items, err := h.service.ListCampaigns(ctx)
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
	h.WritePage(w, r, webtemplates.T(loc, "game.campaigns.title"), http.StatusOK, campaignsListHeader(loc), webtemplates.AppMainLayoutOptions{}, webtemplates.CampaignListFragment(mapCampaignListItems(items, h.now(), loc), loc))
}

// handleStartNewCampaign handles this route in the module transport layer.
func (h handlers) handleStartNewCampaign(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	h.WritePage(w, r,
		webtemplates.T(loc, "game.campaigns.new.title"), http.StatusOK,
		campaignStartHeader(loc),
		webtemplates.AppMainLayoutOptions{},
		webtemplates.CampaignStartFragment(loc),
	)
}

// handleCreateCampaign handles this route in the module transport layer.
func (h handlers) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	h.WritePage(w, r,
		webtemplates.T(loc, "game.create.title"), http.StatusOK,
		campaignCreateHeader(loc),
		webtemplates.AppMainLayoutOptions{},
		webtemplates.CampaignCreateFragment(webtemplates.CampaignCreateFormValues{}, loc),
	)
}

// handleCreateCampaignSubmit handles this route in the module transport layer.
func (h handlers) handleCreateCampaignSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		flash.Write(w, r, flash.Notice{Kind: flash.KindError, Key: "error.web.message.failed_to_parse_campaign_create_form"})
		httpx.WriteRedirect(w, r, routepath.AppCampaignsCreate)
		return
	}
	input, err := parseCreateCampaignInput(r.Form)
	if err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_create_campaign", routepath.AppCampaignsCreate)
		return
	}
	ctx, userID := h.RequestContextAndUserID(r)
	input.Locale = h.RequestLocaleTag(r)
	created, err := h.service.CreateCampaign(ctx, input)
	if err != nil {
		h.writeMutationError(w, r, err, "error.web.message.failed_to_create_campaign", routepath.AppCampaignsCreate)
		return
	}
	if h.sync != nil {
		h.sync.CampaignCreated(ctx, userID, created.CampaignID)
	}

	h.writeMutationSuccess(w, r, "web.campaigns.notice_campaign_created", routepath.AppCampaign(created.CampaignID))
}

// parseAppGameSystem parses inbound values into package-safe forms.
func parseAppGameSystem(value string) (GameSystem, bool) {
	return campaignapp.ParseGameSystem(value)
}

// parseAppGmMode parses inbound values into package-safe forms.
func parseAppGmMode(value string) (GmMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "human":
		return GmModeHuman, true
	case "ai":
		return GmModeAI, true
	case "hybrid":
		return GmModeHybrid, true
	default:
		return GmModeUnspecified, false
	}
}
