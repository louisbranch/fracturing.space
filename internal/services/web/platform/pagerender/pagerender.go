// Package pagerender centralizes module page rendering behavior.
package pagerender

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	flashnotice "github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
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

// emptyComponent defines an internal contract used at this web package boundary.
type emptyComponent struct{}

// Render centralizes this web behavior in one helper seam.
func (emptyComponent) Render(context.Context, io.Writer) error {
	return nil
}

// WriteModulePage writes a module page using shared app-shell rendering contracts.
func WriteModulePage(w http.ResponseWriter, r *http.Request, resolver requestresolver.PageResolver, page ModulePage) error {
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

	pageState := requestresolver.ResolveLocalizedPage(w, r, resolver)
	ctx := httpx.RequestContext(r)
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

	toast := resolveFlashToast(w, r, pageState.Localizer, pageState.Language)
	layout := webtemplates.AppLayoutWithMainHeaderAndLayout(
		page.Title,
		requestresolver.ResolveViewer(r, resolver),
		page.Header,
		page.Layout,
		toast,
		pageState.Language,
		pageState.Localizer,
	)
	if err := layout.Render(templ.WithChildren(ctx, fragment), &buf); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write(buf.Bytes())
	return nil
}

// resolveFlashToast resolves request-scoped values needed by this package.
func resolveFlashToast(w http.ResponseWriter, r *http.Request, loc webi18n.Localizer, _ string) *webtemplates.AppToast {
	notice, ok := flashnotice.ReadAndClear(w, r)
	if !ok {
		return nil
	}
	message := strings.TrimSpace(notice.Message)
	key := strings.TrimSpace(notice.Key)
	if key != "" && loc != nil {
		if localized := strings.TrimSpace(loc.Sprintf(key)); localized != "" && localized != key {
			message = localized
		}
	}
	if message == "" {
		message = key
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
	toast := resolveFlashToast(w, r, nil, "")
	if toast != nil {
		content := body
		body = templ.ComponentFunc(func(ctx context.Context, wr io.Writer) error {
			if err := webtemplates.AppToastComponent(toast).Render(ctx, wr); err != nil {
				return err
			}
			return content.Render(ctx, wr)
		})
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
