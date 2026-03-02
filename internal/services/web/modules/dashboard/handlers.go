package dashboard

import (
	"net/http"

	dashboardapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
)

// dashboardService defines the service contract used by dashboard handlers.
type dashboardService = dashboardapp.Service

// handlers defines an internal contract used at this web package boundary.
type handlers struct {
	modulehandler.Base
	service dashboardService
}

// newHandlers builds package wiring for this web seam.
func newHandlers(s dashboardService, base modulehandler.Base) handlers {
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
	h.WritePage(w, r, webtemplates.T(loc, "dashboard.title"), http.StatusOK, dashboardMainHeader(loc), webtemplates.AppMainLayoutOptions{}, webtemplates.DashboardFragment(mapDashboardTemplateView(view), loc))
}

// dashboardMainHeader centralizes this web behavior in one helper seam.
func dashboardMainHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{Title: webtemplates.T(loc, "dashboard.title")}
}

// mapDashboardTemplateView maps values across transport and domain boundaries.
func mapDashboardTemplateView(view DashboardView) webtemplates.DashboardPageView {
	health := make([]webtemplates.DashboardServiceHealthEntry, len(view.ServiceHealth))
	for i, e := range view.ServiceHealth {
		health[i] = webtemplates.DashboardServiceHealthEntry{Label: e.Label, Available: e.Available}
	}
	return webtemplates.DashboardPageView{
		ProfilePending: webtemplates.DashboardProfilePendingBlock{Visible: view.ShowPendingProfileBlock},
		Adventure:      webtemplates.DashboardAdventureBlock{Visible: view.ShowAdventureBlock},
		ServiceHealth:  health,
	}
}
