package icons

import (
	"net/http"

	"github.com/a-h/templ"
	platformicons "github.com/louisbranch/fracturing.space/internal/platform/icons"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
)

// service implements icons module handlers using shared module dependencies.
type service struct {
	base modulehandler.Base
}

// NewService builds the icons module service implementation.
func NewService(base modulehandler.Base) Service {
	return service{base: base}
}

// HandleIconsPage renders the icons page fragment or full layout.
func (s service) HandleIconsPage(w http.ResponseWriter, r *http.Request) {
	loc, lang := s.base.Localizer(w, r)
	pageCtx := s.base.PageContext(lang, loc, r)
	s.base.RenderPage(
		w,
		r,
		templates.IconsPage(loc),
		templates.IconsFullPage(pageCtx),
		s.base.HTMXLocalizedPageTitle(loc, "title.icons", templates.AppName()),
	)
}

// HandleIconsTable renders the icon catalog table via HTMX.
func (s service) HandleIconsTable(w http.ResponseWriter, r *http.Request) {
	loc, _ := s.base.Localizer(w, r)
	definitions := platformicons.Catalog()
	if len(definitions) == 0 {
		s.renderIconsTable(w, r, nil, loc.Sprintf("icons.empty"), loc)
		return
	}

	rows := buildIconRows(definitions)
	s.renderIconsTable(w, r, rows, "", loc)
}

// renderIconsTable renders an icon catalog table with optional rows and message.
func (s service) renderIconsTable(w http.ResponseWriter, r *http.Request, rows []templates.IconRow, message string, loc *message.Printer) {
	templ.Handler(templates.IconsTable(rows, message, loc)).ServeHTTP(w, r)
}

// buildIconRows formats icon catalog rows for the table.
func buildIconRows(definitions []platformicons.Definition) []templates.IconRow {
	rows := make([]templates.IconRow, 0, len(definitions))
	for _, def := range definitions {
		rows = append(rows, templates.IconRow{
			ID:          def.ID,
			Name:        def.Name,
			Description: def.Description,
			LucideName:  platformicons.LucideNameOrDefault(def.ID),
		})
	}
	return rows
}
