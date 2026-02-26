// Package pagerender centralizes module page rendering behavior.
package pagerender

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// ModulePage describes a module page response for both full-page and HTMX flows.
type ModulePage struct {
	Title      string
	StatusCode int
	Header     *webtemplates.AppMainHeader
	Layout     webtemplates.AppMainLayoutOptions
	Fragment   templ.Component
}

type emptyComponent struct{}

func (emptyComponent) Render(context.Context, io.Writer) error {
	return nil
}

// WriteModulePage writes a module page using shared app-shell rendering contracts.
func WriteModulePage(w http.ResponseWriter, r *http.Request, deps module.Dependencies, page ModulePage) error {
	if w == nil {
		return nil
	}
	statusCode := page.StatusCode
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}
	fragment := page.Fragment
	if fragment == nil {
		fragment = emptyComponent{}
	}

	loc, lang := webi18n.ResolveLocalizer(w, r, deps.ResolveLanguage)
	ctx := requestContext(r)
	viewer := module.Viewer{}
	if deps.ResolveViewer != nil {
		viewer = deps.ResolveViewer(r)
	}
	if httpx.IsHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(statusCode)
		main := webtemplates.AppMainContentWithLayout(page.Header, page.Layout)
		return main.Render(templ.WithChildren(ctx, fragment), w)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	return webtemplates.AppLayoutWithMainHeaderAndLayout(page.Title, viewer, page.Header, page.Layout, lang, loc).Render(templ.WithChildren(ctx, fragment), w)
}

func requestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
