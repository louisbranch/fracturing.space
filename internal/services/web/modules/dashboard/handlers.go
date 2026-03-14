package dashboard

import (
	"net/http"

	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	modulehandler.Base
	service dashboardapp.Service
}

// newHandlers builds package wiring for this web seam.
func newHandlers(s dashboardapp.Service, base modulehandler.Base) handlers {
	return handlers{Base: base, service: s}
}

// handleIndex handles this route in the module transport layer.
func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	ctx, userID := h.RequestContextAndUserID(r)
	view, err := h.service.LoadDashboard(ctx, userID, h.RequestLocaleTag(r))
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.WritePage(w, r, webtemplates.T(loc, "dashboard.title"), http.StatusOK, dashboardMainHeader(loc), webtemplates.AppMainLayoutOptions{}, DashboardFragment(mapDashboardTemplateView(view), loc))
}

// dashboardMainHeader centralizes this web behavior in one helper seam.
func dashboardMainHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{Title: webtemplates.T(loc, "dashboard.title")}
}
