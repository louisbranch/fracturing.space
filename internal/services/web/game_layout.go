package web

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

const gamePageContentType = "text/html; charset=utf-8"

func (h *handler) resolvedAppName() string {
	if h == nil {
		return branding.AppName
	}
	appName := strings.TrimSpace(h.config.AppName)
	if appName == "" {
		return branding.AppName
	}
	return appName
}

func (h *handler) pageContext(w http.ResponseWriter, r *http.Request) webtemplates.PageContext {
	printer, lang := localizer(w, r)
	return webtemplates.PageContext{
		Lang:         lang,
		Loc:          printer,
		CurrentPath:  r.URL.Path,
		CurrentQuery: r.URL.RawQuery,
		AppName:      h.resolvedAppName(),
	}
}

func writeGameContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", gamePageContentType)
}
