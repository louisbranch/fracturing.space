// Package routepath stores canonical HTTP paths for web modules.
package routepath

import (
	"net/url"
	"strings"
)

const (
	Root                                     = "/"
	Login                                    = "/login"
	Logout                                   = "/logout"
	Health                                   = "/up"
	AuthPrefix                               = "/auth/"
	AuthLogin                                = "/auth/login"
	PasskeysPrefix                           = "/passkeys/"
	PasskeyRegisterStart                     = "/passkeys/register/start"
	PasskeyRegisterFinish                    = "/passkeys/register/finish"
	PasskeyLoginStart                        = "/passkeys/login/start"
	PasskeyLoginFinish                       = "/passkeys/login/finish"
	DiscoverPrefix                           = "/discover/"
	UserProfilePrefix                        = "/u/"
	UserProfilePattern                       = UserProfilePrefix + "{username}"
	UserProfilePatternWithTrailingSlash      = UserProfilePrefix + "{username}/"
	UserProfileRestPattern                   = UserProfilePrefix + "{username}/{rest...}"
	AppPrefix                                = "/app/"
	AppDashboard                             = "/app/dashboard"
	DashboardPrefix                          = "/app/dashboard/"
	AppCampaigns                             = "/app/campaigns"
	AppCampaignsNew                          = "/app/campaigns/new"
	AppCampaignsCreate                       = "/app/campaigns/create"
	CampaignsPrefix                          = "/app/campaigns/"
	AppCampaignPattern                       = CampaignsPrefix + "{campaignID}"
	AppCampaignRestPattern                   = CampaignsPrefix + "{campaignID}/{rest...}"
	AppCampaignSessionsPattern               = CampaignsPrefix + "{campaignID}/sessions"
	AppCampaignSessionPattern                = CampaignsPrefix + "{campaignID}/sessions/{sessionID}"
	AppCampaignSessionStartPattern           = CampaignsPrefix + "{campaignID}/sessions/start"
	AppCampaignSessionEndPattern             = CampaignsPrefix + "{campaignID}/sessions/end"
	AppCampaignParticipantsPattern           = CampaignsPrefix + "{campaignID}/participants"
	AppCampaignParticipantUpdatePattern      = CampaignsPrefix + "{campaignID}/participants/update"
	AppCampaignCharactersPattern             = CampaignsPrefix + "{campaignID}/characters"
	AppCampaignCharacterPattern              = CampaignsPrefix + "{campaignID}/characters/{characterID}"
	AppCampaignCharacterCreatePattern        = CampaignsPrefix + "{campaignID}/characters/create"
	AppCampaignCharacterUpdatePattern        = CampaignsPrefix + "{campaignID}/characters/update"
	AppCampaignCharacterControlPattern       = CampaignsPrefix + "{campaignID}/characters/control"
	AppCampaignCharacterCreationStepPattern  = CampaignsPrefix + "{campaignID}/characters/{characterID}/creation/step"
	AppCampaignCharacterCreationResetPattern = CampaignsPrefix + "{campaignID}/characters/{characterID}/creation/reset"
	AppCampaignGamePattern                   = CampaignsPrefix + "{campaignID}/game"
	AppCampaignInvitesPattern                = CampaignsPrefix + "{campaignID}/invites"
	AppCampaignInviteCreatePattern           = CampaignsPrefix + "{campaignID}/invites/create"
	AppCampaignInviteRevokePattern           = CampaignsPrefix + "{campaignID}/invites/revoke"
	AppNotifications                         = "/app/notifications"
	Notifications                            = "/app/notifications/"
	AppNotificationPattern                   = Notifications + "{notificationID}"
	AppNotificationOpenPattern               = Notifications + "{notificationID}/open"
	AppNotificationRestPattern               = Notifications + "{notificationID}/{rest...}"
	AppSettings                              = "/app/settings"
	SettingsPrefix                           = "/app/settings/"
	AppSettingsAIKeyRevokePattern            = SettingsPrefix + "ai-keys/{credentialID}/revoke"
	AppSettingsRestPattern                   = SettingsPrefix + "{rest...}"
	SettingsNoticeQueryKey                   = "notice"
	SettingsNoticePublicProfileRequired      = "public-profile-required"
)

// UserProfile returns the public user profile route.
func UserProfile(username string) string {
	return UserProfilePrefix + escapeSegment(username)
}

// AppCampaign returns the campaign overview route.
func AppCampaign(campaignID string) string {
	return CampaignsPrefix + escapeSegment(campaignID)
}

// AppCampaignSessions returns the campaign sessions route.
func AppCampaignSessions(campaignID string) string {
	return AppCampaign(campaignID) + "/sessions"
}

// AppCampaignSessionStart returns the campaign session-start route.
func AppCampaignSessionStart(campaignID string) string {
	return AppCampaignSessions(campaignID) + "/start"
}

// AppCampaignSessionEnd returns the campaign session-end route.
func AppCampaignSessionEnd(campaignID string) string {
	return AppCampaignSessions(campaignID) + "/end"
}

// AppCampaignSession returns the campaign session-detail route.
func AppCampaignSession(campaignID string, sessionID string) string {
	return AppCampaignSessions(campaignID) + "/" + escapeSegment(sessionID)
}

// AppCampaignParticipants returns the campaign participants route.
func AppCampaignParticipants(campaignID string) string {
	return AppCampaign(campaignID) + "/participants"
}

// AppCampaignParticipantUpdate returns the campaign participant-update route.
func AppCampaignParticipantUpdate(campaignID string) string {
	return AppCampaignParticipants(campaignID) + "/update"
}

// AppCampaignCharacters returns the campaign characters route.
func AppCampaignCharacters(campaignID string) string {
	return AppCampaign(campaignID) + "/characters"
}

// AppCampaignGame returns the campaign game route.
func AppCampaignGame(campaignID string) string {
	return AppCampaign(campaignID) + "/game"
}

// AppCampaignChat returns the legacy campaign chat route alias.
func AppCampaignChat(campaignID string) string {
	return AppCampaignGame(campaignID)
}

// AppCampaignCharacter returns the campaign character-detail route.
func AppCampaignCharacter(campaignID string, characterID string) string {
	return AppCampaignCharacters(campaignID) + "/" + escapeSegment(characterID)
}

// AppCampaignCharacterCreationStep returns the character creation step route.
func AppCampaignCharacterCreationStep(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/creation/step"
}

// AppCampaignCharacterCreationReset returns the character creation reset route.
func AppCampaignCharacterCreationReset(campaignID string, characterID string) string {
	return AppCampaignCharacter(campaignID, characterID) + "/creation/reset"
}

// AppCampaignCharacterCreate returns the campaign character-create route.
func AppCampaignCharacterCreate(campaignID string) string {
	return AppCampaignCharacters(campaignID) + "/create"
}

// AppCampaignCharacterUpdate returns the campaign character-update route.
func AppCampaignCharacterUpdate(campaignID string) string {
	return AppCampaignCharacters(campaignID) + "/update"
}

// AppCampaignCharacterControl returns the campaign character-control route.
func AppCampaignCharacterControl(campaignID string) string {
	return AppCampaignCharacters(campaignID) + "/control"
}

// AppCampaignInvites returns the campaign invites route.
func AppCampaignInvites(campaignID string) string {
	return AppCampaign(campaignID) + "/invites"
}

// AppCampaignInviteCreate returns the campaign invite-create route.
func AppCampaignInviteCreate(campaignID string) string {
	return AppCampaignInvites(campaignID) + "/create"
}

// AppCampaignInviteRevoke returns the campaign invite-revoke route.
func AppCampaignInviteRevoke(campaignID string) string {
	return AppCampaignInvites(campaignID) + "/revoke"
}

// AppNotification returns the notification detail route.
func AppNotification(notificationID string) string {
	return Notifications + escapeSegment(notificationID)
}

// AppNotificationsOpen returns the notification detail route.
func AppNotificationsOpen(notificationID string) string {
	return AppNotification(notificationID)
}

// AppNotificationOpen returns the notification open-and-acknowledge route.
func AppNotificationOpen(notificationID string) string {
	return AppNotification(notificationID) + "/open"
}

// AppSettingsProfile returns the public profile settings route.
const AppSettingsProfile = "/app/settings/profile"

// AppSettingsLocale returns the locale settings route.
const AppSettingsLocale = "/app/settings/locale"

// AppSettingsAIKeys returns the AI keys settings route.
const AppSettingsAIKeys = "/app/settings/ai-keys"

// AppSettingsAIKeyRevoke returns the AI key revoke route.
func AppSettingsAIKeyRevoke(credentialID string) string {
	return AppSettingsAIKeys + "/" + escapeSegment(credentialID) + "/revoke"
}

// AppSettingsProfileWithNotice returns the profile settings route with a notice code.
func AppSettingsProfileWithNotice(notice string) string {
	notice = strings.TrimSpace(notice)
	if notice == "" {
		return AppSettingsProfile
	}
	return AppSettingsProfile + "?" + SettingsNoticeQueryKey + "=" + url.QueryEscape(notice)
}

func escapeSegment(raw string) string {
	return url.PathEscape(strings.TrimSpace(raw))
}
