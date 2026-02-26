package web

import (
	"net/http"

	authfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/auth"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
	feature_discovery "github.com/louisbranch/fracturing.space/internal/services/web/feature/discovery"
	feature_invites "github.com/louisbranch/fracturing.space/internal/services/web/feature/invites"
	feature_notifications "github.com/louisbranch/fracturing.space/internal/services/web/feature/notifications"
	feature_profile "github.com/louisbranch/fracturing.space/internal/services/web/feature/profile"
	feature_publicprofile "github.com/louisbranch/fracturing.space/internal/services/web/feature/publicprofile"
	feature_settings "github.com/louisbranch/fracturing.space/internal/services/web/feature/settings"
)

type appRouteHandlers struct {
	AppHome               http.HandlerFunc
	ProfileHandlers       feature_profile.Handlers
	SettingsHandlers      feature_settings.Handlers
	CampaignHandlers      campaignfeature.Handlers
	InvitesHandlers       feature_invites.Handlers
	NotificationsHandlers feature_notifications.Handlers
}

type publicRouteHandlers struct {
	PublicAuthHandlers    authfeature.PublicHandlers
	PublicProfileHandlers feature_publicprofile.Handlers
	DiscoveryHandlers     feature_discovery.Handlers
}

func (h *handler) appRouteHandlers() appRouteHandlers {
	return h.appRouteHandlersImpl()
}

func (h *handler) publicRouteHandlers() publicRouteHandlers {
	return h.publicRouteHandlersImpl()
}

func (h *handler) campaignFeatureDependencies() campaignfeature.AppCampaignDependencies {
	return h.campaignFeatureDependenciesImpl()
}

func (h *handler) authFlowDependencies() authfeature.AuthFlowDependencies {
	return h.authFlowDependenciesImpl()
}
