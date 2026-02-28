// Package pagerender centralizes module page rendering behavior.
package pagerender

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// RequestResolver resolves viewer and language state from a request.
// This decouples platform rendering from the module-layer Dependencies type.
type RequestResolver interface {
	ResolveRequestViewer(r *http.Request) module.Viewer
	ResolveRequestLanguage(r *http.Request) string
}

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
func WriteModulePage(w http.ResponseWriter, r *http.Request, resolver RequestResolver, page ModulePage) error {
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

	var resolveLanguage module.ResolveLanguage
	if resolver != nil {
		resolveLanguage = resolver.ResolveRequestLanguage
	}
	loc, lang := webi18n.ResolveLocalizer(w, r, resolveLanguage)
	ctx := httpx.RequestContext(r)
	viewer := module.Viewer{}
	if resolver != nil {
		viewer = resolver.ResolveRequestViewer(r)
	}
	var buf bytes.Buffer
	if httpx.IsHTMXRequest(r) {
		main := webtemplates.AppMainContentWithLayout(page.Header, page.Layout)
		if err := main.Render(templ.WithChildren(ctx, fragment), &buf); err != nil {
			return err
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(statusCode)
		_, _ = w.Write(buf.Bytes())
		return nil
	}

	toast := resolveFlashToast(w, r, loc)
	layout := webtemplates.AppLayoutWithMainHeaderAndLayout(page.Title, viewer, page.Header, page.Layout, toast, lang, loc)
	if err := layout.Render(templ.WithChildren(ctx, fragment), &buf); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write(buf.Bytes())
	return nil
}

func resolveFlashToast(w http.ResponseWriter, r *http.Request, loc webi18n.Localizer) *webtemplates.AppToast {
	notice, ok := flashnotice.ReadAndClear(w, r)
	if !ok {
		return nil
	}
	message := strings.TrimSpace(loc.Sprintf(notice.Key))
	if message == "" {
		message = strings.TrimSpace(notice.Key)
	}
	if message == "" {
		return nil
	}
	return &webtemplates.AppToast{
		Kind:    string(notice.Kind),
		Message: message,
	}
}

// WritePublicPage writes a public (unauthenticated) page using the auth layout.
func WritePublicPage(w http.ResponseWriter, r *http.Request, title string, metaDesc string, lang string, statusCode int, body templ.Component) {
	if w == nil {
		return
	}
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}
	if body == nil {
		body = emptyComponent{}
	}

	ctx := templ.WithChildren(httpx.RequestContext(r), body)
	path := ""
	query := ""
	if r != nil && r.URL != nil {
		path = r.URL.Path
		query = r.URL.RawQuery
	}
	var rendered bytes.Buffer
	if err := webtemplates.AuthLayout(title, metaDesc, lang, path, query).Render(ctx, &rendered); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write(rendered.Bytes())
}
