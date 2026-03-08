// Package weberror renders shared app-shell error responses for web modules.
package weberror

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/pagerender"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// ShouldRenderAppError reports whether status should use app error-page UX.
func ShouldRenderAppError(statusCode int) bool {
	return statusCode >= http.StatusBadRequest
}

// PublicMessage resolves a user-safe localized error message.
func PublicMessage(loc webi18n.Localizer, err error) string {
	if err == nil {
		return ""
	}
	if loc != nil {
		if key := apperrors.LocalizationKey(err); key != "" {
			if localized := strings.TrimSpace(loc.Sprintf(key)); localized != "" {
				return localized
			}
		}
	}
	statusCode := apperrors.HTTPStatus(err)
	if statusCode < http.StatusBadRequest {
		statusCode = http.StatusInternalServerError
	}
	if text := strings.TrimSpace(http.StatusText(statusCode)); text != "" {
		return text
	}
	return http.StatusText(http.StatusInternalServerError)
}

// WriteAppError writes a localized app-shell error response for full-page and HTMX requests.
func WriteAppError(w http.ResponseWriter, r *http.Request, statusCode int, resolver pagerender.RequestResolver) {
	if w == nil {
		return
	}

	var resolveLanguage module.ResolveLanguage
	if resolver != nil {
		resolveLanguage = resolver.ResolveRequestLanguage
	}
	loc, lang := webi18n.ResolveLocalizer(w, r, resolveLanguage)
	fragment := webtemplates.AppErrorState(statusCode, loc)

	if httpx.IsHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(statusCode)
		content := webtemplates.AppMainContentWithLayout(nil, webtemplates.AppMainLayoutOptions{})
		if err := content.Render(templ.WithChildren(httpx.RequestContext(r), fragment), w); err != nil {
			http.Error(w, PublicMessage(loc, err), statusCode)
		}
		return
	}

	viewer := module.Viewer{}
	if resolver != nil {
		viewer = resolver.ResolveRequestViewer(r)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	title := webtemplates.AppErrorPageTitle(statusCode, loc)
	if err := webtemplates.AppLayoutWithMainHeaderAndLayout(title, viewer, nil, webtemplates.AppMainLayoutOptions{}, nil, lang, loc).Render(templ.WithChildren(httpx.RequestContext(r), fragment), w); err != nil {
		http.Error(w, PublicMessage(loc, err), statusCode)
	}
}

// WritePublicAppError writes a localized public-shell error page for routes
// outside the authenticated app chrome.
func WritePublicAppError(w http.ResponseWriter, r *http.Request, statusCode int) {
	if w == nil {
		return
	}
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
}

// WritePublicError writes a public-route-safe localized error response.
func WritePublicError(w http.ResponseWriter, r *http.Request, err error) {
	if w == nil {
		return
	}
	WritePublicAppError(w, r, apperrors.HTTPStatus(err))
}

// WriteModuleError writes a module-safe localized error response.
func WriteModuleError(w http.ResponseWriter, r *http.Request, err error, resolver pagerender.RequestResolver) {
	if w == nil {
		return
	}
	WriteAppError(w, r, apperrors.HTTPStatus(err), resolver)
}
