package web

import (
	"net/http"

	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	feature_notifications "github.com/louisbranch/fracturing.space/internal/services/web/feature/notifications"
	featuresettings "github.com/louisbranch/fracturing.space/internal/services/web/feature/settings"
)

// test-only handlers provide a compatibility seam for existing test code while
// production code stays function-first at the route boundary.

func (h *handler) handleAppRoot(w http.ResponseWriter, r *http.Request) {
	h.publicRouteHandlers().PublicAuthHandlers.Root(w, r)
}

func (h *handler) handleAppHome(w http.ResponseWriter, r *http.Request) {
	h.appRouteHandlers().AppHome(w, r)
}

func (h *handler) handleAppDashboard(w http.ResponseWriter, r *http.Request) {
	h.handleAppHome(w, r)
}

func (h *handler) handleAppProfile(w http.ResponseWriter, r *http.Request) {
	h.appRouteHandlers().ProfileHandlers.Profile(w, r)
}

func (h *handler) handleAppSettings(w http.ResponseWriter, r *http.Request) {
	h.appRouteHandlers().SettingsHandlers.Settings(w, r)
}

func (h *handler) handleAppSettingsRoutes(w http.ResponseWriter, r *http.Request) {
	featuresettings.HandleSettingsSubpath(w, r, featuresettings.NewService(h.appRouteHandlers().SettingsHandlers))
}

func (h *handler) handleAppAIKeys(w http.ResponseWriter, r *http.Request) {
	h.appRouteHandlers().SettingsHandlers.AIKeys(w, r)
}

func (h *handler) handleAppAIKeyRevoke(w http.ResponseWriter, r *http.Request, credentialID string) {
	h.appRouteHandlers().SettingsHandlers.AIKeyRevoke(w, r, credentialID)
}

func (h *handler) handleAppCampaigns(w http.ResponseWriter, r *http.Request) {
	h.appRouteHandlers().CampaignHandlers.Campaigns(w, r)
}

func (h *handler) handleAppCampaignCreate(w http.ResponseWriter, r *http.Request) {
	h.appRouteHandlers().CampaignHandlers.CampaignCreate(w, r)
}

func (h *handler) handleAppCampaignDetail(w http.ResponseWriter, r *http.Request) {
	campaignfeature.HandleAppCampaignDetail(h.campaignFeatureDependencies(), w, r)
}

func (h *handler) handleAppInvites(w http.ResponseWriter, r *http.Request) {
	h.appRouteHandlers().InvitesHandlers.Invites(w, r)
}

func (h *handler) handleAppInviteClaim(w http.ResponseWriter, r *http.Request) {
	h.appRouteHandlers().InvitesHandlers.InviteClaim(w, r)
}

func (h *handler) handleAppNotifications(w http.ResponseWriter, r *http.Request) {
	h.appRouteHandlers().NotificationsHandlers.Notifications(w, r)
}

func (h *handler) handleAppNotificationsRoutes(w http.ResponseWriter, r *http.Request) {
	feature_notifications.HandleNotificationSubpath(w, r, feature_notifications.NewService(h.appRouteHandlers().NotificationsHandlers))
}

func (h *handler) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	h.publicRouteHandlers().PublicAuthHandlers.AuthCallback(w, r)
}

func (h *handler) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	h.publicRouteHandlers().PublicAuthHandlers.AuthLogout(w, r)
}

func (h *handler) handlePasskeyRegisterStart(w http.ResponseWriter, r *http.Request) {
	h.publicRouteHandlers().PublicAuthHandlers.PasskeyRegisterStart(w, r)
}
