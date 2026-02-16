package web

import (
	"net/http"

	"github.com/a-h/templ"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

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
