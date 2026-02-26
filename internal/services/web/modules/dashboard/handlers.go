package dashboard

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/pagerender"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/webctx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/weberror"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

type handlers struct {
	service service
	deps    runtimeDependencies
}

type runtimeDependencies struct {
	resolveLanguage module.ResolveLanguage
	resolveViewer   module.ResolveViewer
	resolveUserID   module.ResolveUserID
}

func newRuntimeDependencies(deps module.Dependencies) runtimeDependencies {
	return runtimeDependencies{
		resolveLanguage: deps.ResolveLanguage,
		resolveViewer:   deps.ResolveViewer,
		resolveUserID:   deps.ResolveUserID,
	}
}

func (d runtimeDependencies) moduleDependencies() module.Dependencies {
	return module.Dependencies{
		ResolveViewer:   d.resolveViewer,
		ResolveLanguage: d.resolveLanguage,
		ResolveUserID:   d.resolveUserID,
	}
}

func newHandlers(s service, deps module.Dependencies) handlers {
	return handlers{service: s, deps: newRuntimeDependencies(deps)}
}

func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, lang := h.pageLocalizer(w, r)
	ctx, userID := h.requestContextAndUserID(r)
	view, err := h.service.loadDashboard(ctx, userID, requestLocale(lang))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writePage(w, r, webtemplates.T(loc, "dashboard.title"), dashboardMainHeader(loc), webtemplates.DashboardFragment(mapDashboardTemplateView(view), loc))
}

func dashboardMainHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{Title: webtemplates.T(loc, "dashboard.title")}
}

func mapDashboardTemplateView(view DashboardView) webtemplates.DashboardPageView {
	return webtemplates.DashboardPageView{
		ProfilePending: webtemplates.DashboardProfilePendingBlock{Visible: view.ShowPendingProfileBlock},
	}
}

func requestLocale(languageTag string) commonv1.Locale {
	locale, _ := platformi18n.ParseLocale(languageTag)
	return platformi18n.NormalizeLocale(locale)
}

func (h handlers) writePage(w http.ResponseWriter, r *http.Request, title string, header *webtemplates.AppMainHeader, fragment templ.Component) {
	if err := pagerender.WriteModulePage(w, r, h.deps.moduleDependencies(), pagerender.ModulePage{
		Title:    title,
		Header:   header,
		Fragment: fragment,
	}); err != nil {
		h.writeError(w, r, err)
	}
}

func (h handlers) pageLocalizer(w http.ResponseWriter, r *http.Request) (webtemplates.Localizer, string) {
	loc, lang := webi18n.ResolveLocalizer(w, r, h.deps.resolveLanguage)
	return loc, lang
}

func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	weberror.WriteAppError(w, r, http.StatusNotFound, h.deps.moduleDependencies())
}

func (h handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	weberror.WriteModuleError(w, r, err, h.deps.moduleDependencies())
}

func (h handlers) requestUserID(r *http.Request) string {
	if r == nil || h.deps.resolveUserID == nil {
		return ""
	}
	return strings.TrimSpace(h.deps.resolveUserID(r))
}

func (h handlers) requestContextAndUserID(r *http.Request) (context.Context, string) {
	ctx := webctx.WithResolvedUserID(r, h.deps.resolveUserID)
	return ctx, h.requestUserID(r)
}
