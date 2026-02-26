package web

import (
	"net/http"

	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	feature_invites "github.com/louisbranch/fracturing.space/internal/services/web/feature/invites"
	feature_notifications "github.com/louisbranch/fracturing.space/internal/services/web/feature/notifications"
	feature_profile "github.com/louisbranch/fracturing.space/internal/services/web/feature/profile"
	feature_settings "github.com/louisbranch/fracturing.space/internal/services/web/feature/settings"
)

func (h *handler) buildAppProfileRouteMux(routes feature_profile.Handlers) *http.ServeMux {
	featureMux := http.NewServeMux()
	feature_profile.RegisterRoutes(featureMux, feature_profile.NewService(routes))
	return featureMux
}

func (h *handler) buildAppSettingsRouteMux(routes feature_settings.Handlers) *http.ServeMux {
	featureMux := http.NewServeMux()
	feature_settings.RegisterRoutes(featureMux, feature_settings.NewService(routes))
	return featureMux
}

func (h *handler) buildAppCampaignRouteMux(routes campaignfeature.Handlers) *http.ServeMux {
	featureMux := http.NewServeMux()
	campaignfeature.RegisterRoutes(featureMux, campaignfeature.NewService(routes))
	return featureMux
}

func (h *handler) buildAppInvitesRouteMux(routes feature_invites.Handlers) *http.ServeMux {
	featureMux := http.NewServeMux()
	feature_invites.RegisterRoutes(featureMux, feature_invites.NewService(routes))
	return featureMux
}

func (h *handler) buildAppNotificationsRouteMux(routes feature_notifications.Handlers) *http.ServeMux {
	featureMux := http.NewServeMux()
	feature_notifications.RegisterRoutes(featureMux, feature_notifications.NewService(routes))
	return featureMux
}
