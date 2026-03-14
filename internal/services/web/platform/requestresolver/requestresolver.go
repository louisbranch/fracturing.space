package requestresolver

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
)

// ViewerResolver resolves request-scoped app-chrome viewer state.
type ViewerResolver interface {
	ResolveRequestViewer(*http.Request) module.Viewer
}

// PageResolver resolves the request-scoped viewer and language values used by
// shared page rendering and error helpers.
type PageResolver interface {
	ViewerResolver
	ResolveRequestLanguage(*http.Request) string
}

// PrincipalResolver extends the shared page resolver with signed-in state,
// user-id resolution, and auth gating used by root composition.
type PrincipalResolver interface {
	PageResolver
	ResolveSignedIn(*http.Request) bool
	ResolveUserID(*http.Request) string
	AuthRequired(*http.Request) bool
}

// Base provides a nil-safe implementation of the shared page resolver
// contract so higher-level transport bases can embed it instead of repeating
// the same request-scoped plumbing.
type Base struct {
	resolveLanguage module.ResolveLanguage
	resolveViewer   module.ResolveViewer
}

// Principal provides a nil-safe implementation of the full request-scoped
// principal contract shared by server, composition, and module wiring.
type Principal struct {
	Base
	authRequired    func(*http.Request) bool
	resolveSignedIn module.ResolveSignedIn
	resolveUserID   module.ResolveUserID
}

// LocalizedPage carries the request-scoped localized shell values shared by
// app-shell rendering and error helpers.
type LocalizedPage struct {
	Localizer webi18n.Localizer
	Language  string
}

// New builds a shared request resolver from explicit callbacks.
func New(resolveLanguage module.ResolveLanguage, resolveViewer module.ResolveViewer) Base {
	return Base{
		resolveLanguage: resolveLanguage,
		resolveViewer:   resolveViewer,
	}
}

// NewFromPageResolver builds a shared page resolver base from another page
// resolver contract and preserves nil-safe behavior for callers that accept an
// interface at composition time.
func NewFromPageResolver(resolver PageResolver) Base {
	if resolver == nil {
		return Base{}
	}
	return New(resolver.ResolveRequestLanguage, resolver.ResolveRequestViewer)
}

// WithLanguage returns a copy with the supplied language resolver.
func (b Base) WithLanguage(resolveLanguage module.ResolveLanguage) Base {
	b.resolveLanguage = resolveLanguage
	return b
}

// WithViewer returns a copy with the supplied viewer resolver.
func (b Base) WithViewer(resolveViewer module.ResolveViewer) Base {
	b.resolveViewer = resolveViewer
	return b
}

// NewPrincipal builds a full request-scoped principal resolver from explicit
// callbacks.
func NewPrincipal(
	authRequired func(*http.Request) bool,
	resolveSignedIn module.ResolveSignedIn,
	resolveUserID module.ResolveUserID,
	resolveLanguage module.ResolveLanguage,
	resolveViewer module.ResolveViewer,
) Principal {
	return Principal{
		Base:            New(resolveLanguage, resolveViewer),
		authRequired:    authRequired,
		resolveSignedIn: resolveSignedIn,
		resolveUserID:   resolveUserID,
	}
}

// ResolveLocalizedPage resolves the localized page values shared by shell
// rendering and error helpers.
func ResolveLocalizedPage(w http.ResponseWriter, r *http.Request, resolver PageResolver) LocalizedPage {
	var resolveLanguage module.ResolveLanguage
	if resolver != nil {
		resolveLanguage = resolver.ResolveRequestLanguage
	}
	loc, lang := webi18n.ResolveLocalizer(w, r, resolveLanguage)
	return LocalizedPage{
		Localizer: loc,
		Language:  lang,
	}
}

// ResolveViewer nil-safely resolves viewer chrome state for full-page renders.
func ResolveViewer(r *http.Request, resolver ViewerResolver) module.Viewer {
	if resolver == nil {
		return module.Viewer{}
	}
	return resolver.ResolveRequestViewer(r)
}

// ResolveRequestViewer resolves viewer state for the request.
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

// ResolveSignedIn reports whether the request is associated with a signed-in
// actor.
func (p Principal) ResolveSignedIn(r *http.Request) bool {
	if p.resolveSignedIn == nil {
		return false
	}
	return p.resolveSignedIn(r)
}

// ResolveUserID resolves the authenticated user id for the request.
func (p Principal) ResolveUserID(r *http.Request) string {
	if p.resolveUserID == nil {
		return ""
	}
	return p.resolveUserID(r)
}

// AuthRequired reports whether the request requires authenticated app access.
func (p Principal) AuthRequired(r *http.Request) bool {
	if p.authRequired == nil {
		return false
	}
	return p.authRequired(r)
}
