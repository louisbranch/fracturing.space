// Package publichandler provides a shared base for unauthenticated web module handlers.
// It centralizes error handling, localization, and page rendering that would
// otherwise be duplicated across public modules.
package publichandler

import (
	"net/http"

	"github.com/a-h/templ"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/pagerender"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// Base provides shared error handling and page rendering for public (unauthenticated)
// modules. Embed this in handler structs to get WritePublicPage, WriteNotFound,
// WriteError, and optional viewer resolution for free.
type Base struct {
	resolveViewer         module.ResolveViewer
	resolveViewerSignedIn module.ResolveSignedIn
}

// Option configures a Base.
type Option func(*Base)

// WithResolveViewer attaches a viewer resolver for app-chrome rendering.
func WithResolveViewer(rv module.ResolveViewer) Option {
	return func(b *Base) { b.resolveViewer = rv }
}

// WithResolveViewerSignedIn attaches a direct signed-in resolver to avoid coupling
// auth state to public profile metadata.
func WithResolveViewerSignedIn(resolver module.ResolveSignedIn) Option {
	return func(b *Base) { b.resolveViewerSignedIn = resolver }
}

// NewBase builds a public handler base with the given options.
func NewBase(opts ...Option) Base {
	var b Base
	for _, o := range opts {
		o(&b)
	}
	return b
}

// ResolveRequestViewer resolves viewer state for the request.
// Returns a zero Viewer when no resolver is configured.
func (b Base) ResolveRequestViewer(r *http.Request) module.Viewer {
	if b.resolveViewer == nil {
		return module.Viewer{}
	}
	return b.resolveViewer(r)
}

// IsViewerSignedIn reports whether the current request is authenticated.
func (b Base) IsViewerSignedIn(r *http.Request) bool {
	if b.resolveViewerSignedIn != nil {
		return b.resolveViewerSignedIn(r)
	}
	return false
}

// WritePublicPage renders a full public page using the auth layout.
func (Base) WritePublicPage(w http.ResponseWriter, r *http.Request, title string, metaDesc string, lang string, statusCode int, body templ.Component) {
	pagerender.WritePublicPage(w, r, title, metaDesc, lang, statusCode, body)
}

// WriteNotFound renders a localized 404 error page using the public layout.
func (Base) WriteNotFound(w http.ResponseWriter, r *http.Request) {
	loc, lang := webi18n.ResolveLocalizer(w, r, nil)
	pagerender.WritePublicPage(
		w,
		r,
		webtemplates.AppErrorPageTitle(http.StatusNotFound, loc),
		webtemplates.T(loc, "layout.meta_description"),
		lang,
		http.StatusNotFound,
		webtemplates.AppErrorState(http.StatusNotFound, loc),
	)
}

// WriteError renders a user-safe error response: app error pages for not-found
// and server errors, plain-text status messages for everything else.
func (Base) WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if w == nil {
		return
	}
	statusCode := apperrors.HTTPStatus(err)
	if weberror.ShouldRenderAppError(statusCode) {
		loc, lang := webi18n.ResolveLocalizer(w, r, nil)
		pagerender.WritePublicPage(
			w,
			r,
			webtemplates.AppErrorPageTitle(statusCode, loc),
			webtemplates.T(loc, "layout.meta_description"),
			lang,
			statusCode,
			webtemplates.AppErrorState(statusCode, loc),
		)
		return
	}
	loc, _ := webi18n.ResolveLocalizer(w, r, nil)
	http.Error(w, weberror.PublicMessage(loc, err), statusCode)
}
