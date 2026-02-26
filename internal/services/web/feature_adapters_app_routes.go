package web

import (
	"net/http"

	appfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/app"
)

func (h *handler) appRouteHandlersImpl() appRouteHandlers {
	dependencies := h.appRouteDependencies()
	return appRouteHandlers{
		AppHome: func(w http.ResponseWriter, r *http.Request) {
			appfeature.HandleAppHome(dependencies.appHomeDependencies, w, r)
		},
		ProfileHandlers:       h.appProfileRouteHandlers(),
		SettingsHandlers:      h.appSettingsRouteHandlers(),
		CampaignHandlers:      h.appCampaignRouteHandlers(),
		InvitesHandlers:       h.appInvitesRouteHandlers(),
		NotificationsHandlers: h.appNotificationsRouteHandlers(),
	}
}
