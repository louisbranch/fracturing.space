package admin

import (
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformicons "github.com/louisbranch/fracturing.space/internal/platform/icons"
	catalogmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/catalog"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) handleCampaignsTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.localizer(w, r)
	campaignClient := h.campaignClient()
	if campaignClient == nil {
		h.renderCampaignTable(w, r, nil, loc.Sprintf("error.campaign_service_unavailable"), loc)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := campaignClient.ListCampaigns(ctx, &statev1.ListCampaignsRequest{})
	if err != nil {
		log.Printf("list campaigns: %v", err)
		h.renderCampaignTable(w, r, nil, loc.Sprintf("error.campaigns_unavailable"), loc)
		return
	}

	campaigns := response.GetCampaigns()
	if len(campaigns) == 0 {
		h.renderCampaignTable(w, r, nil, loc.Sprintf("error.no_campaigns"), loc)
		return
	}

	rows := buildCampaignRows(campaigns, loc)
	h.renderCampaignTable(w, r, rows, "", loc)
}

// handleCampaignsPage renders the campaigns page fragment or full layout.
func (h *Handler) handleCampaignsPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.CampaignsPage(loc),
		templates.CampaignsFullPage(pageCtx),
		htmxLocalizedPageTitle(loc, "title.campaigns", templates.AppName()),
	)
}

// handleSystemsPage renders the systems page fragment or full layout.
func (h *Handler) handleSystemsPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.SystemsPage(loc),
		templates.SystemsFullPage(pageCtx),
		htmxLocalizedPageTitle(loc, "title.systems", templates.AppName()),
	)
}

// handleIconsPage renders the icons page fragment or full layout.
func (h *Handler) handleIconsPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.IconsPage(loc),
		templates.IconsFullPage(pageCtx),
		htmxLocalizedPageTitle(loc, "title.icons", templates.AppName()),
	)
}

// handleCatalogPage renders the catalog page fragment or full layout.
func (h *Handler) handleCatalogPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	sectionID := templates.DefaultDaggerheartCatalogSection()
	renderPage(
		w,
		r,
		templates.CatalogPage(sectionID, loc),
		templates.CatalogFullPage(sectionID, pageCtx),
		htmxLocalizedPageTitle(loc, "title.catalog", templates.AppName()),
	)
}

func (h *Handler) handleCatalogSection(w http.ResponseWriter, r *http.Request, sectionID string) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	var full templ.Component
	if !isHTMXRequest(r) {
		full = templates.CatalogFullPage(sectionID, pageCtx)
	}
	renderPage(
		w,
		r,
		templates.CatalogSectionPanel(sectionID, loc),
		full,
		htmxLocalizedPageTitle(loc, "title.catalog", templates.AppName()),
	)
}

func (h *Handler) handleCatalogSectionTable(w http.ResponseWriter, r *http.Request, sectionID string) {
	loc, lang := h.localizer(w, r)
	columns := catalogSectionColumns(sectionID, loc)
	view := templates.CatalogTableView{
		SectionID:   sectionID,
		Columns:     columns,
		Message:     loc.Sprintf("catalog.loading"),
		HrefBaseURL: routepath.CatalogSection(catalogmodule.DaggerheartSystemID, sectionID),
		HTMXBaseURL: routepath.CatalogSectionTable(catalogmodule.DaggerheartSystemID, sectionID),
	}

	contentClient := h.daggerheartContentClient()
	if contentClient == nil {
		view.Message = loc.Sprintf("catalog.error.service_unavailable")
		templ.Handler(templates.CatalogTable(view, loc)).ServeHTTP(w, r)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	pageToken := r.URL.Query().Get("page_token")
	locale := localeFromTag(lang)
	message := ""
	var nextToken, prevToken string
	var rows []templates.CatalogTableRow

	switch sectionID {
	case templates.CatalogSectionClasses:
		resp, err := contentClient.ListClasses(ctx, &daggerheartv1.ListDaggerheartClassesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list classes: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogClassRows(resp.GetClasses())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionSubclasses:
		resp, err := contentClient.ListSubclasses(ctx, &daggerheartv1.ListDaggerheartSubclassesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list subclasses: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogSubclassRows(resp.GetSubclasses())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionHeritages:
		resp, err := contentClient.ListHeritages(ctx, &daggerheartv1.ListDaggerheartHeritagesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list heritages: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogHeritageRows(resp.GetHeritages())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionExperiences:
		resp, err := contentClient.ListExperiences(ctx, &daggerheartv1.ListDaggerheartExperiencesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list experiences: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogExperienceRows(resp.GetExperiences())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionDomains:
		resp, err := contentClient.ListDomains(ctx, &daggerheartv1.ListDaggerheartDomainsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list domains: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogDomainRows(resp.GetDomains())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionDomainCards:
		resp, err := contentClient.ListDomainCards(ctx, &daggerheartv1.ListDaggerheartDomainCardsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "level",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list domain cards: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogDomainCardRows(resp.GetDomainCards())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionItems:
		resp, err := contentClient.ListItems(ctx, &daggerheartv1.ListDaggerheartItemsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list items: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogItemRows(resp.GetItems())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionWeapons:
		resp, err := contentClient.ListWeapons(ctx, &daggerheartv1.ListDaggerheartWeaponsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list weapons: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogWeaponRows(resp.GetWeapons())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionArmor:
		resp, err := contentClient.ListArmor(ctx, &daggerheartv1.ListDaggerheartArmorRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list armor: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogArmorRows(resp.GetArmor())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionLoot:
		resp, err := contentClient.ListLootEntries(ctx, &daggerheartv1.ListDaggerheartLootEntriesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "roll",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list loot: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogLootRows(resp.GetEntries())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionDamageTypes:
		resp, err := contentClient.ListDamageTypes(ctx, &daggerheartv1.ListDaggerheartDamageTypesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list damage types: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogDamageTypeRows(resp.GetDamageTypes())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionAdversaries:
		resp, err := contentClient.ListAdversaries(ctx, &daggerheartv1.ListDaggerheartAdversariesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list adversaries: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogAdversaryRows(resp.GetAdversaries())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionBeastforms:
		resp, err := contentClient.ListBeastforms(ctx, &daggerheartv1.ListDaggerheartBeastformsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list beastforms: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogBeastformRows(resp.GetBeastforms())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionCompanionExperiences:
		resp, err := contentClient.ListCompanionExperiences(ctx, &daggerheartv1.ListDaggerheartCompanionExperiencesRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list companion experiences: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogCompanionExperienceRows(resp.GetExperiences())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	case templates.CatalogSectionEnvironments:
		resp, err := contentClient.ListEnvironments(ctx, &daggerheartv1.ListDaggerheartEnvironmentsRequest{
			PageSize:  catalogListPageSize,
			PageToken: pageToken,
			OrderBy:   "name",
			Locale:    locale,
		})
		if err != nil {
			log.Printf("list environments: %v", err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else if resp != nil {
			rows = buildCatalogEnvironmentRows(resp.GetEnvironments())
			nextToken = resp.GetNextPageToken()
			prevToken = resp.GetPreviousPageToken()
		}
	default:
		message = loc.Sprintf("catalog.error.entries_unavailable")
	}

	if len(rows) == 0 && message == "" {
		message = loc.Sprintf("catalog.empty")
	}

	view.Rows = rows
	view.Message = message
	view.NextToken = nextToken
	view.PrevToken = prevToken

	templ.Handler(templates.CatalogTable(view, loc)).ServeHTTP(w, r)
}

func (h *Handler) handleCatalogSectionDetail(w http.ResponseWriter, r *http.Request, sectionID, entryID string) {
	loc, lang := h.localizer(w, r)
	pageCtx := h.pageContext(lang, loc, r)
	contentClient := h.daggerheartContentClient()
	if contentClient == nil {
		view := templates.CatalogDetailView{
			SectionID: sectionID,
			Title:     templates.DaggerheartCatalogSectionLabel(loc, sectionID),
			Message:   loc.Sprintf("catalog.error.service_unavailable"),
			BackURL:   routepath.CatalogSection(catalogmodule.DaggerheartSystemID, sectionID),
		}
		full := templates.CatalogFullPageWithContent(sectionID, templates.CatalogDetailPanel(view, loc), pageCtx)
		if isHTMXRequest(r) {
			full = nil
		}
		renderPage(
			w,
			r,
			templates.CatalogDetailPanel(view, loc),
			full,
			htmxLocalizedPageTitle(loc, "title.catalog", templates.AppName()),
		)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	locale := localeFromTag(lang)
	view := templates.CatalogDetailView{
		SectionID: sectionID,
		BackURL:   routepath.CatalogSection(catalogmodule.DaggerheartSystemID, sectionID),
	}

	switch sectionID {
	case templates.CatalogSectionClasses:
		resp, err := contentClient.GetClass(ctx, &daggerheartv1.GetDaggerheartClassRequest{Id: entryID, Locale: locale})
		view = buildCatalogClassDetail(sectionID, entryID, resp.GetClass(), err, loc)
	case templates.CatalogSectionSubclasses:
		resp, err := contentClient.GetSubclass(ctx, &daggerheartv1.GetDaggerheartSubclassRequest{Id: entryID, Locale: locale})
		view = buildCatalogSubclassDetail(sectionID, entryID, resp.GetSubclass(), err, loc)
	case templates.CatalogSectionHeritages:
		resp, err := contentClient.GetHeritage(ctx, &daggerheartv1.GetDaggerheartHeritageRequest{Id: entryID, Locale: locale})
		view = buildCatalogHeritageDetail(sectionID, entryID, resp.GetHeritage(), err, loc)
	case templates.CatalogSectionExperiences:
		resp, err := contentClient.GetExperience(ctx, &daggerheartv1.GetDaggerheartExperienceRequest{Id: entryID, Locale: locale})
		view = buildCatalogExperienceDetail(sectionID, entryID, resp.GetExperience(), err, loc)
	case templates.CatalogSectionDomains:
		resp, err := contentClient.GetDomain(ctx, &daggerheartv1.GetDaggerheartDomainRequest{Id: entryID, Locale: locale})
		view = buildCatalogDomainDetail(sectionID, entryID, resp.GetDomain(), err, loc)
	case templates.CatalogSectionDomainCards:
		resp, err := contentClient.GetDomainCard(ctx, &daggerheartv1.GetDaggerheartDomainCardRequest{Id: entryID, Locale: locale})
		view = buildCatalogDomainCardDetail(sectionID, entryID, resp.GetDomainCard(), err, loc)
	case templates.CatalogSectionItems:
		resp, err := contentClient.GetItem(ctx, &daggerheartv1.GetDaggerheartItemRequest{Id: entryID, Locale: locale})
		view = buildCatalogItemDetail(sectionID, entryID, resp.GetItem(), err, loc)
	case templates.CatalogSectionWeapons:
		resp, err := contentClient.GetWeapon(ctx, &daggerheartv1.GetDaggerheartWeaponRequest{Id: entryID, Locale: locale})
		view = buildCatalogWeaponDetail(sectionID, entryID, resp.GetWeapon(), err, loc)
	case templates.CatalogSectionArmor:
		resp, err := contentClient.GetArmor(ctx, &daggerheartv1.GetDaggerheartArmorRequest{Id: entryID, Locale: locale})
		view = buildCatalogArmorDetail(sectionID, entryID, resp.GetArmor(), err, loc)
	case templates.CatalogSectionLoot:
		resp, err := contentClient.GetLootEntry(ctx, &daggerheartv1.GetDaggerheartLootEntryRequest{Id: entryID, Locale: locale})
		view = buildCatalogLootDetail(sectionID, entryID, resp.GetEntry(), err, loc)
	case templates.CatalogSectionDamageTypes:
		resp, err := contentClient.GetDamageType(ctx, &daggerheartv1.GetDaggerheartDamageTypeRequest{Id: entryID, Locale: locale})
		view = buildCatalogDamageTypeDetail(sectionID, entryID, resp.GetDamageType(), err, loc)
	case templates.CatalogSectionAdversaries:
		resp, err := contentClient.GetAdversary(ctx, &daggerheartv1.GetDaggerheartAdversaryRequest{Id: entryID, Locale: locale})
		view = buildCatalogAdversaryDetail(sectionID, entryID, resp.GetAdversary(), err, loc)
	case templates.CatalogSectionBeastforms:
		resp, err := contentClient.GetBeastform(ctx, &daggerheartv1.GetDaggerheartBeastformRequest{Id: entryID, Locale: locale})
		view = buildCatalogBeastformDetail(sectionID, entryID, resp.GetBeastform(), err, loc)
	case templates.CatalogSectionCompanionExperiences:
		resp, err := contentClient.GetCompanionExperience(ctx, &daggerheartv1.GetDaggerheartCompanionExperienceRequest{Id: entryID, Locale: locale})
		view = buildCatalogCompanionExperienceDetail(sectionID, entryID, resp.GetExperience(), err, loc)
	case templates.CatalogSectionEnvironments:
		resp, err := contentClient.GetEnvironment(ctx, &daggerheartv1.GetDaggerheartEnvironmentRequest{Id: entryID, Locale: locale})
		view = buildCatalogEnvironmentDetail(sectionID, entryID, resp.GetEnvironment(), err, loc)
	default:
		view.Title = templates.DaggerheartCatalogSectionLabel(loc, sectionID)
		view.Message = loc.Sprintf("catalog.error.not_found")
		view.BackURL = routepath.CatalogSection(catalogmodule.DaggerheartSystemID, sectionID)
	}

	var full templ.Component
	if !isHTMXRequest(r) {
		full = templates.CatalogFullPageWithContent(sectionID, templates.CatalogDetailPanel(view, loc), pageCtx)
	}
	renderPage(
		w,
		r,
		templates.CatalogDetailPanel(view, loc),
		full,
		htmxLocalizedPageTitle(loc, "title.catalog", templates.AppName()),
	)
}

// handleSystemsTable renders the systems table via HTMX.
func (h *Handler) handleSystemsTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.localizer(w, r)
	client := h.systemClient()
	if client == nil {
		h.renderSystemsTable(w, r, nil, loc.Sprintf("error.system_service_unavailable"), loc)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := client.ListGameSystems(ctx, &statev1.ListGameSystemsRequest{})
	if err != nil {
		log.Printf("list game systems: %v", err)
		h.renderSystemsTable(w, r, nil, loc.Sprintf("error.systems_unavailable"), loc)
		return
	}

	systemsList := response.GetSystems()
	if len(systemsList) == 0 {
		h.renderSystemsTable(w, r, nil, loc.Sprintf("error.no_systems"), loc)
		return
	}

	rows := buildSystemRows(systemsList, loc)
	h.renderSystemsTable(w, r, rows, "", loc)
}

// handleIconsTable renders the icon catalog table via HTMX.
func (h *Handler) handleIconsTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.localizer(w, r)
	definitions := platformicons.Catalog()
	if len(definitions) == 0 {
		h.renderIconsTable(w, r, nil, loc.Sprintf("icons.empty"), loc)
		return
	}

	rows := buildIconRows(definitions)
	h.renderIconsTable(w, r, rows, "", loc)
}

// handleSystemDetail renders the system detail page.
func (h *Handler) handleSystemDetail(w http.ResponseWriter, r *http.Request, systemID string) {
	loc, lang := h.localizer(w, r)
	client := h.systemClient()
	if client == nil {
		h.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_service_unavailable"), lang, loc)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	version := strings.TrimSpace(r.URL.Query().Get("version"))
	parsedID := parseSystemID(systemID)
	if parsedID == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		h.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_not_found"), lang, loc)
		return
	}
	response, err := client.GetGameSystem(ctx, &statev1.GetGameSystemRequest{
		Id:      parsedID,
		Version: version,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			h.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_not_found"), lang, loc)
			return
		}
		log.Printf("get game system: %v", err)
		h.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_unavailable"), lang, loc)
		return
	}
	if response.GetSystem() == nil {
		h.renderSystemDetail(w, r, templates.SystemDetail{}, loc.Sprintf("error.system_not_found"), lang, loc)
		return
	}

	detail := buildSystemDetail(response.GetSystem(), loc)
	h.renderSystemDetail(w, r, detail, "", lang, loc)
}

// handleCampaignDetail renders the single-campaign detail content.
func (h *Handler) handleCampaignDetail(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignClient := h.campaignClient()
	if campaignClient == nil {
		h.renderCampaignDetail(w, r, templates.CampaignDetail{}, loc.Sprintf("error.campaign_service_unavailable"), lang, loc)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := campaignClient.GetCampaign(ctx, &statev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		log.Printf("get campaign: %v", err)
		h.renderCampaignDetail(w, r, templates.CampaignDetail{}, loc.Sprintf("error.campaign_unavailable"), lang, loc)
		return
	}

	campaign := response.GetCampaign()
	if campaign == nil {
		h.renderCampaignDetail(w, r, templates.CampaignDetail{}, loc.Sprintf("error.campaign_not_found"), lang, loc)
		return
	}

	detail := buildCampaignDetail(campaign, loc)
	h.renderCampaignDetail(w, r, detail, "", lang, loc)
}

// handleSessionsList renders the sessions list page.
func (h *Handler) handleSessionsList(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, lang := h.localizer(w, r)
	campaignName := getCampaignName(h, r, campaignID, loc)
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.SessionsListPage(campaignID, campaignName, loc),
		templates.SessionsListFullPage(campaignID, campaignName, pageCtx),
		htmxLocalizedPageTitle(loc, "title.sessions", templates.AppName()),
	)
}

// handleSessionsTable renders the sessions table via HTMX.
func (h *Handler) handleSessionsTable(w http.ResponseWriter, r *http.Request, campaignID string) {
	loc, _ := h.localizer(w, r)
	sessionClient := h.sessionClient()
	if sessionClient == nil {
		h.renderCampaignSessions(w, r, nil, loc.Sprintf("error.session_service_unavailable"), loc)
		return
	}

	ctx, cancel := h.gameGRPCCallContext(r.Context())
	defer cancel()

	response, err := sessionClient.ListSessions(ctx, &statev1.ListSessionsRequest{
		CampaignId: campaignID,
		PageSize:   sessionListPageSize,
	})
	if err != nil {
		log.Printf("list sessions: %v", err)
		h.renderCampaignSessions(w, r, nil, loc.Sprintf("error.sessions_unavailable"), loc)
		return
	}

	sessions := response.GetSessions()
	if len(sessions) == 0 {
		h.renderCampaignSessions(w, r, nil, loc.Sprintf("error.no_sessions"), loc)
		return
	}

	rows := buildCampaignSessionRows(sessions, loc)
	h.renderCampaignSessions(w, r, rows, "", loc)
}

// renderCampaignTable renders a campaign table with optional rows and message.
func (h *Handler) renderCampaignTable(w http.ResponseWriter, r *http.Request, rows []templates.CampaignRow, message string, loc *message.Printer) {
	templ.Handler(templates.CampaignsTable(rows, message, loc)).ServeHTTP(w, r)
}

// renderCampaignDetail renders the campaign detail fragment or full layout.
func (h *Handler) renderCampaignDetail(w http.ResponseWriter, r *http.Request, detail templates.CampaignDetail, message string, lang string, loc *message.Printer) {
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.CampaignDetailPage(detail, message, loc),
		templates.CampaignDetailFullPage(detail, message, pageCtx),
		htmxLocalizedPageTitle(loc, "title.campaign", templates.AppName()),
	)
}

// renderSystemsTable renders a systems table with optional rows and message.
func (h *Handler) renderSystemsTable(w http.ResponseWriter, r *http.Request, rows []templates.SystemRow, message string, loc *message.Printer) {
	templ.Handler(templates.SystemsTable(rows, message, loc)).ServeHTTP(w, r)
}

// renderIconsTable renders an icon catalog table with optional rows and message.
func (h *Handler) renderIconsTable(w http.ResponseWriter, r *http.Request, rows []templates.IconRow, message string, loc *message.Printer) {
	templ.Handler(templates.IconsTable(rows, message, loc)).ServeHTTP(w, r)
}

// renderSystemDetail renders the system detail fragment or full layout.
func (h *Handler) renderSystemDetail(w http.ResponseWriter, r *http.Request, detail templates.SystemDetail, message string, lang string, loc *message.Printer) {
	pageCtx := h.pageContext(lang, loc, r)
	renderPage(
		w,
		r,
		templates.SystemDetailPage(detail, message, loc),
		templates.SystemDetailFullPage(detail, message, pageCtx),
		htmxLocalizedPageTitle(loc, "title.system", templates.AppName()),
	)
}

// renderCampaignSessions renders the session list fragment.
func (h *Handler) renderCampaignSessions(w http.ResponseWriter, r *http.Request, rows []templates.CampaignSessionRow, message string, loc *message.Printer) {
	templ.Handler(templates.CampaignSessionsList(rows, message, loc)).ServeHTTP(w, r)
}

// renderUsersTable renders the users table component.
func (h *Handler) renderUsersTable(w http.ResponseWriter, r *http.Request, rows []templates.UserRow, message string, loc *message.Printer) {
	templ.Handler(templates.UsersTable(rows, message, loc)).ServeHTTP(w, r)
}
