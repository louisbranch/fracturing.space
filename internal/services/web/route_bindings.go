package web

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

func (h *handler) registerGameRoutes(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	appRoutes := h.appRouteHandlersImpl()
	settingsMux := h.buildAppSettingsRouteMux(appRoutes.SettingsHandlers)
	campaignMux := h.buildAppCampaignRouteMux(appRoutes.CampaignHandlers)
	invitesMux := h.buildAppInvitesRouteMux(appRoutes.InvitesHandlers)
	notificationsMux := h.buildAppNotificationsRouteMux(appRoutes.NotificationsHandlers)

	mux.HandleFunc(routepath.AppRoot, appRoutes.AppHome)
	mux.HandleFunc(routepath.AppRootPrefix, appRoutes.AppHome)
	mux.Handle(routepath.AppProfile, h.buildAppProfileRouteMux(appRoutes.ProfileHandlers))
	mux.Handle(routepath.AppSettings, settingsMux)
	mux.Handle(routepath.AppSettingsPrefix, settingsMux)
	mux.Handle(routepath.AppCampaigns, campaignMux)
	mux.Handle(routepath.AppCampaignsPrefix, campaignMux)
	mux.Handle(routepath.AppInvites, invitesMux)
	mux.Handle(routepath.AppInviteClaim, invitesMux)
	mux.Handle(routepath.AppNotifications, notificationsMux)
	mux.Handle(routepath.AppNotificationsPrefix, notificationsMux)
}

func (h *handler) registerPublicRoutes(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	publicRoutes := h.publicRouteHandlersImpl()
	authMux := h.buildPublicAuthRouteMux(publicRoutes.PublicAuthHandlers)
	profileMux := h.buildPublicProfileRouteMux(publicRoutes.PublicProfileHandlers)
	discoveryMux := h.buildPublicDiscoveryRouteMux(publicRoutes.DiscoveryHandlers)

	mux.Handle(routepath.Root, authMux)
	mux.Handle(routepath.Login, authMux)
	mux.Handle(routepath.AuthLogin, authMux)
	mux.Handle(routepath.AuthCallback, authMux)
	mux.Handle(routepath.AuthLogout, authMux)
	mux.Handle(routepath.MagicLink, authMux)
	mux.Handle(routepath.PasskeyRegisterStart, authMux)
	mux.Handle(routepath.PasskeyRegisterFinish, authMux)
	mux.Handle(routepath.PasskeyLoginStart, authMux)
	mux.Handle(routepath.PasskeyLoginFinish, authMux)
	mux.Handle(routepath.Health, authMux)
	mux.Handle(routepath.UserProfilePrefix, profileMux)
	mux.Handle(routepath.Discover, discoveryMux)
	mux.Handle(routepath.DiscoverCampaignsPrefix, discoveryMux)
}
