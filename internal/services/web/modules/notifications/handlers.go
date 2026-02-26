package notifications

import (
	"net/http"
	"strings"

	"github.com/a-h/templ"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/pagerender"
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
}

func newRuntimeDependencies(deps module.Dependencies) runtimeDependencies {
	return runtimeDependencies{
		resolveLanguage: deps.ResolveLanguage,
		resolveViewer:   deps.ResolveViewer,
	}
}

func (d runtimeDependencies) moduleDependencies() module.Dependencies {
	return module.Dependencies{
		ResolveViewer:   d.resolveViewer,
		ResolveLanguage: d.resolveLanguage,
	}
}

func newHandlers(s service, deps module.Dependencies) handlers {
	return handlers{service: s, deps: newRuntimeDependencies(deps)}
}

func (h handlers) handleOpenRoute(w http.ResponseWriter, r *http.Request) {
	notificationID := strings.TrimSpace(r.PathValue("notificationID"))
	if notificationID == "" {
		h.handleNotFound(w, r)
		return
	}
	h.handleOpen(w, r, notificationID)
}

func (h handlers) handleNotFound(w http.ResponseWriter, r *http.Request) {
	weberror.WriteAppError(w, r, http.StatusNotFound, h.deps.moduleDependencies())
}

func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.listNotifications(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if len(items) == 0 {
		weberror.WriteAppError(w, r, http.StatusNotFound, h.deps.moduleDependencies())
		return
	}
	h.writePage(w, r, "Notifications", webtemplates.ScaffoldPage("notifications-root"))
}

func (h handlers) handleOpen(w http.ResponseWriter, r *http.Request, notificationID string) {
	item, err := h.service.openNotification(r.Context(), notificationID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if item.ID == "" {
		weberror.WriteAppError(w, r, http.StatusNotFound, h.deps.moduleDependencies())
		return
	}
	h.writePage(w, r, "Notification", webtemplates.ScaffoldPage("notification-open"))
}

func (h handlers) writePage(w http.ResponseWriter, r *http.Request, title string, body templ.Component) {
	if err := pagerender.WriteModulePage(w, r, h.deps.moduleDependencies(), pagerender.ModulePage{
		Title:    title,
		Fragment: body,
	}); err != nil {
		h.writeError(w, r, err)
	}
}

func (h handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	weberror.WriteModuleError(w, r, err, h.deps.moduleDependencies())
}
