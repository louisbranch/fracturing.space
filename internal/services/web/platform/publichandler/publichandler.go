// Package publichandler provides a shared base for unauthenticated web module handlers.
// It centralizes error handling, localization, and page rendering that would
// otherwise be duplicated across public modules.
package publichandler

import (
	"net/http"

	"github.com/a-h/templ"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/pagerender"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestresolver"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
)

// Base provides shared error handling and page rendering for public (unauthenticated)
// modules. Embed this in handler structs to get WritePublicPage, WriteNotFound,
// WriteError, and optional viewer resolution for free.
type Base struct {
	requestresolver.Base
	resolveViewerSignedIn module.ResolveSignedIn
}

// Option configures a Base.
type Option func(*Base)

// WithResolveViewer attaches a viewer resolver for app-chrome rendering.
func WithResolveViewer(rv module.ResolveViewer) Option {
	return func(b *Base) { b.Base = b.Base.WithViewer(rv) }
}

// WithResolveViewerSignedIn attaches a direct signed-in resolver to avoid coupling
// auth state to public profile metadata.
func WithResolveViewerSignedIn(resolver module.ResolveSignedIn) Option {
	return func(b *Base) { b.resolveViewerSignedIn = resolver }
}

// NewBase builds a public handler base with the given options.
func NewBase(opts ...Option) Base {
	b := Base{Base: requestresolver.New(nil, nil)}
	for _, o := range opts {
		o(&b)
	}
	return b
}

// NewBaseFromPrincipal builds a public handler base from the shared principal
// resolver seam used by root composition.
func NewBaseFromPrincipal(resolver requestresolver.PrincipalResolver) Base {
	var resolveSignedIn module.ResolveSignedIn
	if resolver != nil {
		resolveSignedIn = resolver.ResolveSignedIn
	}
	return Base{
		Base:                  requestresolver.NewFromPageResolver(resolver),
		resolveViewerSignedIn: resolveSignedIn,
	}
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
	weberror.WritePublicAppError(w, r, http.StatusNotFound)
}

// WriteError renders a user-safe error response: app error pages for not-found
// and server errors, plain-text status messages for everything else.
func (Base) WriteError(w http.ResponseWriter, r *http.Request, err error) {
	weberror.WritePublicError(w, r, err)
}
