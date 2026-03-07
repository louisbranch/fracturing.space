// Package modulehandler provides shared transport helpers for admin modules.
package modulehandler

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/louisbranch/fracturing.space/internal/platform/requestctx"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	sharedhtmx "github.com/louisbranch/fracturing.space/internal/services/shared/htmx"
	"github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"golang.org/x/text/message"
)

const grpcRequestTimeout = timeouts.GRPCRequest

// Base carries shared request-scoped module dependencies.
// It is zero-value usable — no constructor is required.
type Base struct{}

// NewBase returns a shared module handler base.
func NewBase() Base {
	return Base{}
}

// Localizer resolves request localizer and selected language.
func (b Base) Localizer(w http.ResponseWriter, r *http.Request) (*message.Printer, string) {
	tag, persist := i18nhttp.ResolveTag(r)
	if persist {
		i18nhttp.SetLanguageCookie(w, tag)
	}
	return i18nhttp.Printer(tag), tag.String()
}

// PageContext builds common template page context from request state.
func (b Base) PageContext(lang string, loc *message.Printer, r *http.Request) templates.PageContext {
	path := ""
	query := ""
	if r != nil && r.URL != nil {
		path = r.URL.Path
		query = r.URL.RawQuery
	}
	return templates.PageContext{
		Lang:         lang,
		Loc:          loc,
		CurrentPath:  path,
		CurrentQuery: query,
	}
}

// GameGRPCCallContext creates a bounded game RPC context with user identity.
func (b Base) GameGRPCCallContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, grpcRequestTimeout)
	if userID := strings.TrimSpace(requestctx.UserIDFromContext(parent)); userID != "" {
		ctx = grpcauthctx.WithUserID(ctx, userID)
	}
	return ctx, cancel
}

// IsHTMXRequest reports whether the request originated from HTMX.
func (b Base) IsHTMXRequest(r *http.Request) bool {
	return sharedhtmx.IsHTMXRequest(r)
}

// RenderPage applies shared HTMX/full-page rendering behavior.
func (b Base) RenderPage(w http.ResponseWriter, r *http.Request, fragment templ.Component, full templ.Component, htmxTitle string) {
	sharedhtmx.RenderPage(w, r, fragment, full, htmxTitle)
}

// HTMXLocalizedPageTitle returns a localized page title for HTMX swaps.
func (b Base) HTMXLocalizedPageTitle(loc *message.Printer, title string, args ...any) string {
	if loc == nil {
		return sharedhtmx.TitleTag("Admin | " + templates.AppName())
	}
	return sharedhtmx.TitleTag(templates.ComposeAdminPageTitle(templates.T(loc, title, args...)))
}
