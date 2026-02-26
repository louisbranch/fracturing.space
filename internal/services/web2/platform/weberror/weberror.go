// Package weberror renders shared app-shell error responses for web2 modules.
package weberror

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web2/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/httpx"
	web2i18n "github.com/louisbranch/fracturing.space/internal/services/web2/platform/i18n"
	web2templates "github.com/louisbranch/fracturing.space/internal/services/web2/templates"
)

// ShouldRenderAppError reports whether status should use app error-page UX.
func ShouldRenderAppError(statusCode int) bool {
	return statusCode == http.StatusNotFound || statusCode >= http.StatusInternalServerError
}

// PublicMessage resolves a user-safe localized error message.
func PublicMessage(loc web2i18n.Localizer, err error) string {
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
func WriteAppError(w http.ResponseWriter, r *http.Request, statusCode int, deps module.Dependencies) {
	if w == nil {
		return
	}
	if !ShouldRenderAppError(statusCode) {
		statusCode = http.StatusInternalServerError
	}

	loc, lang := web2i18n.ResolveLocalizer(w, r, deps.ResolveLanguage)
	fragment := web2templates.AppErrorState(statusCode, loc)

	if httpx.IsHTMXRequest(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(statusCode)
		content := web2templates.AppMainContentWithLayout(nil, web2templates.AppMainLayoutOptions{})
		if err := content.Render(templ.WithChildren(requestContext(r), fragment), w); err != nil {
			http.Error(w, PublicMessage(loc, err), statusCode)
		}
		return
	}

	viewer := module.Viewer{}
	if deps.ResolveViewer != nil {
		viewer = deps.ResolveViewer(r)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	title := web2templates.AppErrorPageTitle(statusCode, loc)
	if err := web2templates.AppLayoutWithMainHeaderAndLayout(title, viewer, nil, web2templates.AppMainLayoutOptions{}, lang, loc).Render(templ.WithChildren(requestContext(r), fragment), w); err != nil {
		http.Error(w, PublicMessage(loc, err), statusCode)
	}
}

func requestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

// WriteModuleError writes a module-safe localized error response.
func WriteModuleError(w http.ResponseWriter, r *http.Request, err error, deps module.Dependencies) {
	if w == nil {
		return
	}
	statusCode := apperrors.HTTPStatus(err)
	if ShouldRenderAppError(statusCode) {
		WriteAppError(w, r, statusCode, deps)
		return
	}
	loc, _ := web2i18n.ResolveLocalizer(w, r, deps.ResolveLanguage)
	http.Error(w, PublicMessage(loc, err), statusCode)
}
