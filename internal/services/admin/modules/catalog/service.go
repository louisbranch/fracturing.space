package catalog

import (
	"log"
	"net/http"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
)

// service implements the catalog service handlers for module-local routing.
type service struct {
	base modulehandler.Base
}

const (
	// catalogListPageSize caps the number of catalog entries shown per page.
	catalogListPageSize = 25
)

// NewService returns the catalog service backed by shared module handler dependencies.
func NewService(base modulehandler.Base) Service {
	return &service{base: base}
}

// HandleCatalogPage renders the catalog page fragment or full layout.
func (s *service) HandleCatalogPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	loc, lang := s.base.Localizer(w, r)
	pageCtx := s.base.PageContext(lang, loc, r)
	sectionID := templates.DefaultDaggerheartCatalogSection()
	s.base.RenderPage(
		w,
		r,
		templates.CatalogPage(sectionID, loc),
		templates.CatalogFullPage(sectionID, pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.catalog", templates.AppName()),
	)
}

// HandleCatalogSection renders the catalog section panel fragment or full layout.
func (s *service) HandleCatalogSection(w http.ResponseWriter, r *http.Request, sectionID string) {
	loc, lang := s.base.Localizer(w, r)
	pageCtx := s.base.PageContext(lang, loc, r)
	var full templ.Component
	if !s.base.IsHTMXRequest(r) {
		full = templates.CatalogFullPage(sectionID, pageCtx)
	}
	s.base.RenderPage(
		w,
		r,
		templates.CatalogSectionPanel(sectionID, loc),
		full,
		s.base.HTMXLocalizedPageTitle(loc, "title.catalog", templates.AppName()),
	)
}

// HandleCatalogSectionTable renders the section table rows and pagination state.
func (s *service) HandleCatalogSectionTable(w http.ResponseWriter, r *http.Request, sectionID string) {
	loc, lang := s.base.Localizer(w, r)
	columns := catalogSectionColumns(sectionID, loc)
	view := templates.CatalogTableView{
		SectionID:   sectionID,
		Columns:     columns,
		Message:     loc.Sprintf("catalog.loading"),
		HrefBaseURL: routepath.CatalogSection(DaggerheartSystemID, sectionID),
		HTMXBaseURL: routepath.CatalogSectionTable(DaggerheartSystemID, sectionID),
	}

	contentClient := s.base.DaggerheartContentClient()
	if contentClient == nil {
		view.Message = loc.Sprintf("catalog.error.service_unavailable")
		templ.Handler(templates.CatalogTable(view, loc)).ServeHTTP(w, r)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	pageToken := ""
	if r != nil && r.URL != nil {
		pageToken = r.URL.Query().Get("page_token")
	}
	locale := localeFromTag(lang)
	message := ""
	var nextToken, prevToken string
	var rows []templates.CatalogTableRow

	loader, ok := catalogSectionTableLoaders[sectionID]
	if !ok {
		message = loc.Sprintf("catalog.error.entries_unavailable")
	} else {
		loadedRows, loadedNextToken, loadedPrevToken, err := loader(ctx, contentClient, pageToken, locale)
		if err != nil {
			log.Printf("list catalog section %s: %v", sectionID, err)
			message = loc.Sprintf("catalog.error.entries_unavailable")
		} else {
			rows = loadedRows
			nextToken = loadedNextToken
			prevToken = loadedPrevToken
		}
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

// HandleCatalogSectionDetail renders detail content for a selected catalog entry.
func (s *service) HandleCatalogSectionDetail(w http.ResponseWriter, r *http.Request, sectionID string, entryID string) {
	loc, lang := s.base.Localizer(w, r)
	pageCtx := s.base.PageContext(lang, loc, r)
	contentClient := s.base.DaggerheartContentClient()
	if contentClient == nil {
		view := templates.CatalogDetailView{
			SectionID: sectionID,
			Title:     templates.DaggerheartCatalogSectionLabel(loc, sectionID),
			Message:   loc.Sprintf("catalog.error.service_unavailable"),
			BackURL:   routepath.CatalogSection(DaggerheartSystemID, sectionID),
		}
		var full templ.Component
		if !s.base.IsHTMXRequest(r) {
			full = templates.CatalogFullPageWithContent(sectionID, templates.CatalogDetailPanel(view, loc), pageCtx)
		}
		s.base.RenderPage(
			w,
			r,
			templates.CatalogDetailPanel(view, loc),
			full,
			s.base.HTMXLocalizedPageTitle(loc, "title.catalog", templates.AppName()),
		)
		return
	}

	ctx, cancel := s.base.GameGRPCCallContext(r.Context())
	defer cancel()

	locale := localeFromTag(lang)
	view := templates.CatalogDetailView{
		SectionID: sectionID,
		BackURL:   routepath.CatalogSection(DaggerheartSystemID, sectionID),
	}

	detailLoader, ok := catalogSectionDetailLoaders[sectionID]
	if !ok {
		view.Title = templates.DaggerheartCatalogSectionLabel(loc, sectionID)
		view.Message = loc.Sprintf("catalog.error.not_found")
		view.BackURL = routepath.CatalogSection(DaggerheartSystemID, sectionID)
	} else {
		view = detailLoader(ctx, contentClient, sectionID, entryID, locale, loc)
	}

	var full templ.Component
	if !s.base.IsHTMXRequest(r) {
		full = templates.CatalogFullPageWithContent(sectionID, templates.CatalogDetailPanel(view, loc), pageCtx)
	}
	s.base.RenderPage(
		w,
		r,
		templates.CatalogDetailPanel(view, loc),
		full,
		s.base.HTMXLocalizedPageTitle(loc, "title.catalog", templates.AppName()),
	)
}

func localeFromTag(tag string) commonv1.Locale {
	if locale, ok := platformi18n.ParseLocale(tag); ok {
		return locale
	}
	return platformi18n.DefaultLocale()
}
