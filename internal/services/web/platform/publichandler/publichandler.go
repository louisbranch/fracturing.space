package publichandler

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/pagerender"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// Base provides shared error handling and page rendering for public (unauthenticated)
// modules. Embed this in handler structs to get WritePublicPage, WriteNotFound,
// WriteError, and optional viewer resolution for free.
type Base struct {
	principal.Base
	resolveViewerSignedIn principal.SignedInFunc
	resolveUserID         principal.UserIDFunc
}

// Option configures a Base.
type Option func(*Base)

// WithResolveViewer attaches a viewer resolver for app-chrome rendering.
func WithResolveViewer(rv principal.ViewerFunc) Option {
	return func(b *Base) { b.Base = b.Base.WithViewer(rv) }
}

// WithResolveViewerSignedIn attaches a direct signed-in resolver to avoid coupling
// auth state to public profile metadata.
func WithResolveViewerSignedIn(resolver principal.SignedInFunc) Option {
	return func(b *Base) { b.resolveViewerSignedIn = resolver }
}

// WithResolveUserID attaches a direct user-id resolver for shared public
// transport flows that need to branch on the signed-in viewer.
func WithResolveUserID(resolver principal.UserIDFunc) Option {
	return func(b *Base) { b.resolveUserID = resolver }
}

// NewBase builds a public handler base with the given options.
func NewBase(opts ...Option) Base {
	b := Base{Base: principal.NewBase(nil, nil)}
	for _, o := range opts {
		o(&b)
	}
	return b
}

// NewBaseFromPrincipal builds a public handler base from the shared principal
// resolver seam used by root composition.
func NewBaseFromPrincipal(resolver principal.PrincipalResolver) Base {
	var resolveSignedIn principal.SignedInFunc
	var resolveUserID principal.UserIDFunc
	if resolver != nil {
		resolveSignedIn = resolver.ResolveSignedIn
		resolveUserID = resolver.ResolveUserID
	}
	return Base{
		Base:                  principal.NewBaseFromPageResolver(resolver),
		resolveViewerSignedIn: resolveSignedIn,
		resolveUserID:         resolveUserID,
	}
}

// IsViewerSignedIn reports whether the current request is authenticated.
func (b Base) IsViewerSignedIn(r *http.Request) bool {
	if b.resolveViewerSignedIn != nil {
		return b.resolveViewerSignedIn(r)
	}
	return false
}

// RequestUserID resolves the signed-in viewer id for public transport flows.
func (b Base) RequestUserID(r *http.Request) string {
	if r == nil || b.resolveUserID == nil {
		return ""
	}
	return strings.TrimSpace(b.resolveUserID(r))
}

// PageLocalizer resolves the shared localized page state used by public pages
// and JSON error responses.
func (b Base) PageLocalizer(w http.ResponseWriter, r *http.Request) (webtemplates.Localizer, string) {
	page := principal.ResolveLocalizedPage(w, r, &b)
	return page.Localizer, page.Language
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
