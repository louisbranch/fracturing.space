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
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"

	"golang.org/x/text/language"
)

// Base carries the shared request-scoped resolvers used by protected module handlers.
// Embed this in module handler structs to get standard user resolution, localization,
// page rendering, and error writing without duplicating boilerplate.
type Base struct {
	principal.Base
	resolveUserID principal.UserIDFunc
}

// NewBase builds a handler base from explicit resolver functions.
func NewBase(resolveUserID principal.UserIDFunc, resolveLanguage principal.LanguageFunc, resolveViewer principal.ViewerFunc) Base {
	return Base{
		Base:          principal.NewBase(resolveLanguage, resolveViewer),
		resolveUserID: resolveUserID,
	}
}

// NewBaseFromPrincipal builds a handler base from the shared principal
// resolver seam used by root composition.
func NewBaseFromPrincipal(resolver principal.PrincipalResolver) Base {
	var resolveUserID principal.UserIDFunc
	if resolver != nil {
		resolveUserID = resolver.ResolveUserID
	}
	return Base{
		Base:          principal.NewBaseFromPageResolver(resolver),
		resolveUserID: resolveUserID,
	}
}

// NewTestBase builds a handler base with no-op resolvers suitable for tests
// that do not exercise user resolution, localization, or viewer state.
func NewTestBase() Base {
	return Base{
		Base: principal.NewBase(
			func(*http.Request) string { return "" },
			func(*http.Request) module.Viewer { return module.Viewer{} },
		),
		resolveUserID: func(*http.Request) string { return "" },
	}
}

// PageLocalizer resolves a localizer and language tag from the request.
func (b Base) PageLocalizer(w http.ResponseWriter, r *http.Request) (webtemplates.Localizer, string) {
	page := principal.ResolveLocalizedPage(w, r, &b)
	return page.Localizer, page.Language
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
	return webi18n.ResolveTag(r, b.ResolveRequestLanguage)
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
