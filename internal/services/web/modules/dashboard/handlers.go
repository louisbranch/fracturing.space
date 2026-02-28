package dashboard

import (
	"context"
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/language"
)

// dashboardService defines the service operations used by dashboard handlers.
type dashboardService interface {
	loadDashboard(ctx context.Context, userID string, locale language.Tag) (DashboardView, error)
}

type handlers struct {
	modulehandler.Base
	service dashboardService
}

func newHandlers(s service, base modulehandler.Base) handlers {
	return handlers{Base: base, service: s}
}

func (h handlers) handleIndex(w http.ResponseWriter, r *http.Request) {
	loc, _ := h.PageLocalizer(w, r)
	ctx, userID := h.RequestContextAndUserID(r)
	view, err := h.service.loadDashboard(ctx, userID, h.RequestLocaleTag(r))
	if err != nil {
		h.WriteError(w, r, err)
		return
	}
	h.WritePage(w, r, webtemplates.T(loc, "dashboard.title"), http.StatusOK, dashboardMainHeader(loc), webtemplates.AppMainLayoutOptions{}, webtemplates.DashboardFragment(mapDashboardTemplateView(view), loc))
}

func dashboardMainHeader(loc webtemplates.Localizer) *webtemplates.AppMainHeader {
	return &webtemplates.AppMainHeader{Title: webtemplates.T(loc, "dashboard.title")}
}

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
