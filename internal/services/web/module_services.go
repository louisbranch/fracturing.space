package web

import (
	"net/http"

	authmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/auth"
	campaignsmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/campaigns"
	discoverymodule "github.com/louisbranch/fracturing.space/internal/services/web/module/discovery"
	invitesmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/invites"
	notificationsmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/notifications"
	profilemodule "github.com/louisbranch/fracturing.space/internal/services/web/module/profile"
	publicprofilemodule "github.com/louisbranch/fracturing.space/internal/services/web/module/publicprofile"
	settingsmodule "github.com/louisbranch/fracturing.space/internal/services/web/module/settings"
)

func buildProfileModuleService(h *handler) profilemodule.Service {
	if h == nil {
		return nil
	}
	return profilemodule.NewService(profilemodule.Handlers{
		Profile: h.handleAppProfile,
	})
}

func buildSettingsModuleService(h *handler) settingsmodule.Service {
	if h == nil {
		return nil
	}
	return settingsmodule.NewService(settingsmodule.Handlers{
		Settings:            h.handleAppSettings,
		UserProfileSettings: h.handleAppUserProfileSettings,
		AIKeys:              h.handleAppAIKeys,
		AIKeyRevoke:         h.handleAppAIKeyRevoke,
	})
}

func buildCampaignModuleService(h *handler) campaignsmodule.Service {
	if h == nil {
		return nil
	}
	return campaignsmodule.NewService(campaignsmodule.Handlers{
		Campaigns:                 h.handleAppCampaigns,
		CampaignCreate:            h.handleAppCampaignCreate,
		CampaignOverview:          h.handleAppCampaignOverview,
		CampaignSessions:          h.handleAppCampaignSessions,
		CampaignSessionStart:      h.handleAppCampaignSessionStart,
		CampaignSessionEnd:        h.handleAppCampaignSessionEnd,
		CampaignSessionDetail:     h.handleAppCampaignSessionDetail,
		CampaignParticipants:      h.handleAppCampaignParticipants,
		CampaignParticipantUpdate: h.handleAppCampaignParticipantUpdate,
		CampaignCharacters:        h.handleAppCampaignCharacters,
		CampaignCharacterCreate:   h.handleAppCampaignCharacterCreate,
		CampaignCharacterUpdate:   h.handleAppCampaignCharacterUpdate,
		CampaignCharacterControl:  h.handleAppCampaignCharacterControl,
		CampaignCharacterDetail:   h.handleAppCampaignCharacterDetail,
		CampaignInvites:           h.handleAppCampaignInvites,
		CampaignInviteCreate:      h.handleAppCampaignInviteCreate,
		CampaignInviteRevoke:      h.handleAppCampaignInviteRevoke,
	})
}

func buildInvitesModuleService(h *handler) invitesmodule.Service {
	if h == nil {
		return nil
	}
	return invitesmodule.NewService(invitesmodule.Handlers{
		Invites:     h.handleAppInvites,
		InviteClaim: h.handleAppInviteClaim,
	})
}

func buildNotificationsModuleService(h *handler) notificationsmodule.Service {
	if h == nil {
		return nil
	}
	return notificationsmodule.NewService(notificationsmodule.Handlers{
		Notifications:    h.handleAppNotifications,
		NotificationOpen: h.handleAppNotificationOpen,
	})
}

func buildPublicAuthModuleService(h *handler) authmodule.PublicService {
	if h == nil {
		return nil
	}
	appName := h.resolvedAppName()
	return authmodule.NewPublicService(authmodule.PublicHandlers{
		Root: func(w http.ResponseWriter, r *http.Request) {
			h.handleAppRoot(w, r)
		},
		Login: func(w http.ResponseWriter, r *http.Request) {
			h.handleAppLoginPage(w, r, appName)
		},
		AuthLogin:             h.handleAuthLogin,
		AuthCallback:          h.handleAuthCallback,
		AuthLogout:            h.handleAuthLogout,
		MagicLink:             h.handleMagicLink,
		PasskeyRegisterStart:  h.handlePasskeyRegisterStart,
		PasskeyRegisterFinish: h.handlePasskeyRegisterFinish,
		PasskeyLoginStart:     h.handlePasskeyLoginStart,
		PasskeyLoginFinish:    h.handlePasskeyLoginFinish,
		Health: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		},
	})
}

func buildPublicProfileModuleService(h *handler) publicprofilemodule.Service {
	if h == nil {
		return nil
	}
	return publicprofilemodule.NewService(publicprofilemodule.Handlers{
		PublicProfile: h.handlePublicProfile,
	})
}

func buildDiscoveryModuleService(h *handler) discoverymodule.Service {
	if h == nil {
		return nil
	}
	return discoverymodule.NewService(discoverymodule.Handlers{
		Discover:         h.handleDiscover,
		DiscoverCampaign: h.handleDiscoverCampaign,
	})
}
