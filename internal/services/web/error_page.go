package web

import (
	"net/http"

	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// renderErrorPage converts internal transport and auth errors to a localized web
// error template, so failure states stay in one shared UX surface.
func (h *handler) renderErrorPage(w http.ResponseWriter, r *http.Request, status int, title string, message string) {
	printer, lang := localizer(w, r)
	page := webtemplates.PageContext{
		Lang:         lang,
		Loc:          printer,
		CurrentPath:  r.URL.Path,
		CurrentQuery: r.URL.RawQuery,
	}
	w.WriteHeader(status)
	templ.Handler(webtemplates.ErrorPage(page, title, message)).ServeHTTP(w, r)
}
