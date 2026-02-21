package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	sharedhtmx "github.com/louisbranch/fracturing.space/internal/services/shared/htmx"
	sharedtemplates "github.com/louisbranch/fracturing.space/internal/services/shared/templates"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

const gamePageContentType = "text/html; charset=utf-8"

var errNoWebPageComponent = errors.New("web: no page component provided")

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

func (h *handler) pageContextForCampaign(w http.ResponseWriter, r *http.Request, campaignID string) webtemplates.PageContext {
	page := h.pageContext(w, r)
	page.CampaignName = h.campaignDisplayName(r.Context(), campaignID)
	return page
}

func (h *handler) writePage(w http.ResponseWriter, r *http.Request, page templ.Component, htmxTitle string) error {
	return writePage(w, r, page, htmxTitle)
}

func writePage(w http.ResponseWriter, r *http.Request, page templ.Component, htmxTitle string) error {
	writeGameContentType(w)
	if page == nil {
		return errNoWebPageComponent
	}
	if sharedhtmx.IsHTMXRequest(r) {
		sharedhtmx.RenderPage(w, r, page, page, htmxTitle)
		return nil
	}
	return page.Render(r.Context(), w)
}

func composeHTMXTitle(loc webtemplates.Localizer, title string, args ...any) string {
	if loc == nil {
		return sharedhtmx.TitleTag(sharedtemplates.ComposePageTitle(title))
	}
	return sharedhtmx.TitleTag(sharedtemplates.ComposePageTitle(webtemplates.T(loc, title, args...)))
}

func writeGameContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", gamePageContentType)
}
