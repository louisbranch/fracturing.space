// Package modulehandler provides a composable base for protected web module handlers.
//
// Protected modules (those mounted under /app/) share common handler infrastructure
// for user resolution, localization, page rendering, and error handling. This package
// extracts that shared scaffold so modules embed it rather than duplicating it.
package modulehandler

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/pagerender"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/webctx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"

	"golang.org/x/text/language"
)

// Base carries the shared request-scoped resolvers used by protected module handlers.
// Embed this in module handler structs to get standard user resolution, localization,
// page rendering, and error writing without duplicating boilerplate.
type Base struct {
	resolveUserID   module.ResolveUserID
	resolveLanguage module.ResolveLanguage
	resolveViewer   module.ResolveViewer
}

// NewBase builds a handler base from explicit resolver functions.
func NewBase(resolveUserID module.ResolveUserID, resolveLanguage module.ResolveLanguage, resolveViewer module.ResolveViewer) Base {
	return Base{
		resolveUserID:   resolveUserID,
		resolveLanguage: resolveLanguage,
		resolveViewer:   resolveViewer,
	}
}

// NewTestBase builds a handler base with no-op resolvers suitable for tests
// that do not exercise user resolution, localization, or viewer state.
func NewTestBase() Base {
	return Base{
		resolveUserID:   func(*http.Request) string { return "" },
		resolveLanguage: func(*http.Request) string { return "" },
		resolveViewer:   func(*http.Request) module.Viewer { return module.Viewer{} },
	}
}

// ResolveRequestViewer resolves app chrome viewer state for a request.
func (b Base) ResolveRequestViewer(r *http.Request) module.Viewer {
	if b.resolveViewer == nil {
		return module.Viewer{}
	}
	return b.resolveViewer(r)
}

// ResolveRequestLanguage returns the effective request language.
func (b Base) ResolveRequestLanguage(r *http.Request) string {
	if b.resolveLanguage == nil {
		return ""
	}
	return b.resolveLanguage(r)
}

// PageLocalizer resolves a localizer and language tag from the request.
func (b Base) PageLocalizer(w http.ResponseWriter, r *http.Request) (webtemplates.Localizer, string) {
	return webi18n.ResolveLocalizer(w, r, b.resolveLanguage)
}

// WriteError renders a localized module error response.
func (b Base) WriteError(w http.ResponseWriter, r *http.Request, err error) {
	weberror.WriteModuleError(w, r, err, &b)
}

// WriteNotFound renders a 404 error page within the app shell.
func (b Base) WriteNotFound(w http.ResponseWriter, r *http.Request) {
	weberror.WriteAppError(w, r, http.StatusNotFound, &b)
}

// RequestUserID extracts the authenticated user ID from the request.
func (b Base) RequestUserID(r *http.Request) string {
	if r == nil || b.resolveUserID == nil {
		return ""
	}
	return strings.TrimSpace(b.resolveUserID(r))
}

// RequestContextAndUserID returns a context enriched with the user ID (for gRPC
// downstream calls) and the raw user ID string.
func (b Base) RequestContextAndUserID(r *http.Request) (context.Context, string) {
	ctx := webctx.WithResolvedUserID(r, b.resolveUserID)
	return ctx, b.RequestUserID(r)
}

// RequestLocaleTag returns the resolved language tag for the request, suitable
// for locale resolution. Prefer ResolveRequestLanguage for display language and
// RequestContextAndUserID for user-scoped context.
func (b Base) RequestLocaleTag(r *http.Request) language.Tag {
	return webi18n.ResolveTag(r, b.resolveLanguage)
}

// WritePage renders a full module page (HTMX-aware) with the given title, header,
// layout, and content fragment.
func (b Base) WritePage(
	w http.ResponseWriter,
	r *http.Request,
	title string,
	statusCode int,
	header *webtemplates.AppMainHeader,
	layout webtemplates.AppMainLayoutOptions,
	fragment templ.Component,
) {
	if err := pagerender.WriteModulePage(w, r, &b, pagerender.ModulePage{
		Title:      title,
		StatusCode: statusCode,
		Header:     header,
		Layout:     layout,
		Fragment:   fragment,
	}); err != nil {
		b.WriteError(w, r, err)
	}
}
