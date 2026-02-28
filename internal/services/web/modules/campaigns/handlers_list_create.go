package campaigns

import (
	"net/http"
	"strings"

	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// --- Headers ---

func campaignsListHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title: webtemplates.T(loc, "game.campaigns.title"),
		Action: &webtemplates.AppMainHeaderAction{
			Label: webtemplates.T(loc, "game.campaigns.start_new"),
			URL:   routepath.AppCampaignsNew,
		},
	}
}

func campaignStartHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{
		Title: webtemplates.T(loc, "game.campaigns.new.title"),
		Breadcrumbs: []sharedtemplates.BreadcrumbItem{
			{Label: webtemplates.T(loc, "game.campaigns.title"), URL: routepath.AppCampaigns},
			{Label: webtemplates.T(loc, "game.campaigns.new.title")},
		},
	}
}

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

func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	ctx, _ := h.RequestContextAndUserID(r)
	items, err := h.service.listCampaigns(ctx)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.WritePage(w, r, webtemplates.T(loc, "game.campaigns.title"), http.StatusOK, campaignsListHeader(loc), webtemplates.AppMainLayoutOptions{}, webtemplates.CampaignListFragment(mapCampaignListItems(items), loc))
}

func (h handlers) handleStartNewCampaign(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	h.WritePage(w, r,
		webtemplates.T(loc, "game.campaigns.new.title"), http.StatusOK,
		campaignStartHeader(loc),
		webtemplates.AppMainLayoutOptions{},
		webtemplates.CampaignStartFragment(loc),
	)
}

func (h handlers) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	h.WritePage(w, r,
		webtemplates.T(loc, "game.create.title"), http.StatusOK,
		campaignCreateHeader(loc),
		webtemplates.AppMainLayoutOptions{},
		webtemplates.CampaignCreateFragment(webtemplates.CampaignCreateFormValues{}, loc),
	)
}

func (h handlers) handleCreateCampaignSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_parse_campaign_create_form", "failed to parse campaign create form"))
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))

	systemValue := strings.TrimSpace(r.FormValue("system"))
	if systemValue == "" {
		systemValue = "daggerheart"
	}
	system, ok := parseAppGameSystem(systemValue)
	if !ok {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_system_is_invalid", "campaign system is invalid"))
		return
	}

	gmModeValue := strings.TrimSpace(r.FormValue("gm_mode"))
	if gmModeValue == "" {
		gmModeValue = "human"
	}
	gmMode, ok := parseAppGmMode(gmModeValue)
	if !ok {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.campaign_gm_mode_is_invalid", "campaign gm mode is invalid"))
		return
	}

	themePrompt := strings.TrimSpace(r.FormValue("theme_prompt"))
	ctx, _ := h.RequestContextAndUserID(r)

	created, err := h.service.createCampaign(ctx, CreateCampaignInput{
		Name:        name,
		System:      system,
		GMMode:      gmMode,
		ThemePrompt: themePrompt,
		Locale:      h.RequestLocaleTag(r),
	})
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	httpx.WriteRedirect(w, r, routepath.AppCampaign(created.CampaignID))
}

func parseAppGameSystem(value string) (GameSystem, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "daggerheart", "game_system_daggerheart":
		return GameSystemDaggerheart, true
	default:
		return GameSystemUnspecified, false
	}
}

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
