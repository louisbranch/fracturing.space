package routepath

import (
	"net/url"
	"strings"
)

const (
	// Public/auth surface routes.
	Root                    = "/"
	Login                   = "/login"
	AuthLogin               = "/auth/login"
	AuthCallback            = "/auth/callback"
	AuthLogout              = "/auth/logout"
	MagicLink               = "/magic"
	PasskeyRegisterStart    = "/passkeys/register/start"
	PasskeyRegisterFinish   = "/passkeys/register/finish"
	PasskeyLoginStart       = "/passkeys/login/start"
	PasskeyLoginFinish      = "/passkeys/login/finish"
	Health                  = "/up"
	UserProfilePrefix       = "/u/"
	Discover                = "/discover"
	DiscoverPrefix          = "/discover/"
	DiscoverCampaigns       = "/discover/campaigns"
	DiscoverCampaignsPrefix = "/discover/campaigns/"
)

const (
	// Canonical authenticated app surface routes.
	AppRoot                = "/app"
	AppRootPrefix          = "/app/"
	AppCampaigns           = "/app/campaigns"
	AppCampaignsCreate     = "/app/campaigns/create"
	AppCampaignsPrefix     = "/app/campaigns/"
	AppProfile             = "/app/profile"
	AppSettings            = "/app/settings"
	AppSettingsPrefix      = "/app/settings/"
	AppInvites             = "/app/invites"
	AppInviteClaim         = "/app/invites/claim"
	AppNotifications       = "/app/notifications"
	AppNotificationsPrefix = "/app/notifications/"
)

// Campaign returns the campaign overview route.
func Campaign(campaignID string) string {
	return AppCampaigns + "/" + escapeSegment(campaignID)
}

// CampaignSessions returns the campaign sessions route.
func CampaignSessions(campaignID string) string {
	return Campaign(campaignID) + "/sessions"
}

// CampaignSessionStart returns the campaign session-start route.
func CampaignSessionStart(campaignID string) string {
	return CampaignSessions(campaignID) + "/start"
}

// CampaignSessionEnd returns the campaign session-end route.
func CampaignSessionEnd(campaignID string) string {
	return CampaignSessions(campaignID) + "/end"
}

// CampaignSession returns the campaign session-detail route.
func CampaignSession(campaignID string, sessionID string) string {
	return CampaignSessions(campaignID) + "/" + escapeSegment(sessionID)
}

// CampaignParticipants returns the campaign participants route.
func CampaignParticipants(campaignID string) string {
	return Campaign(campaignID) + "/participants"
}

// CampaignParticipantUpdate returns the campaign participant-update route.
func CampaignParticipantUpdate(campaignID string) string {
	return CampaignParticipants(campaignID) + "/update"
}

// CampaignCharacters returns the campaign characters route.
func CampaignCharacters(campaignID string) string {
	return Campaign(campaignID) + "/characters"
}

// CampaignCharacterCreate returns the campaign character-create route.
func CampaignCharacterCreate(campaignID string) string {
	return CampaignCharacters(campaignID) + "/create"
}

// CampaignCharacterUpdate returns the campaign character-update route.
func CampaignCharacterUpdate(campaignID string) string {
	return CampaignCharacters(campaignID) + "/update"
}

// CampaignCharacterControl returns the campaign character-control route.
func CampaignCharacterControl(campaignID string) string {
	return CampaignCharacters(campaignID) + "/control"
}

// CampaignCharacter returns the campaign character-detail route.
func CampaignCharacter(campaignID string, characterID string) string {
	return CampaignCharacters(campaignID) + "/" + escapeSegment(characterID)
}

// CampaignInvites returns the campaign invites route.
func CampaignInvites(campaignID string) string {
	return Campaign(campaignID) + "/invites"
}

// CampaignInviteCreate returns the campaign invite-create route.
func CampaignInviteCreate(campaignID string) string {
	return CampaignInvites(campaignID) + "/create"
}

// CampaignInviteRevoke returns the campaign invite-revoke route.
func CampaignInviteRevoke(campaignID string) string {
	return CampaignInvites(campaignID) + "/revoke"
}

// UserProfile returns the public-profile route.
func UserProfile(username string) string {
	return UserProfilePrefix + escapeSegment(username)
}

// DiscoverCampaign returns the discover campaign-detail route.
func DiscoverCampaign(campaignID string) string {
	return DiscoverCampaigns + "/" + escapeSegment(campaignID)
}

func escapeSegment(raw string) string {
	return url.PathEscape(strings.TrimSpace(raw))
}
