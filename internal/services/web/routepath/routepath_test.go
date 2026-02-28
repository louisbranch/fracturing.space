package routepath

import "testing"

func TestTopLevelRouteConstants(t *testing.T) {
	t.Parallel()

	if Root != "/" {
		t.Fatalf("Root = %q", Root)
	}
	if Login != "/login" {
		t.Fatalf("Login = %q", Login)
	}
	if Logout != "/logout" {
		t.Fatalf("Logout = %q", Logout)
	}
	if Health != "/up" {
		t.Fatalf("Health = %q", Health)
	}
	if AppDashboard != "/app/dashboard" {
		t.Fatalf("AppDashboard = %q", AppDashboard)
	}
	if DashboardPrefix != "/app/dashboard/" {
		t.Fatalf("DashboardPrefix = %q", DashboardPrefix)
	}
	if AppCampaignsNew != "/app/campaigns/new" {
		t.Fatalf("AppCampaignsNew = %q", AppCampaignsNew)
	}
	if UserProfilePrefix != "/u/" {
		t.Fatalf("UserProfilePrefix = %q", UserProfilePrefix)
	}
	if CampaignsPrefix != "/app/campaigns/" {
		t.Fatalf("CampaignsPrefix = %q", CampaignsPrefix)
	}
	if Notifications != "/app/notifications/" {
		t.Fatalf("Notifications = %q", Notifications)
	}
	if SettingsPrefix != "/app/settings/" {
		t.Fatalf("SettingsPrefix = %q", SettingsPrefix)
	}
	if SettingsNoticeQueryKey != "notice" {
		t.Fatalf("SettingsNoticeQueryKey = %q", SettingsNoticeQueryKey)
	}
	if SettingsNoticePublicProfileRequired != "public-profile-required" {
		t.Fatalf("SettingsNoticePublicProfileRequired = %q", SettingsNoticePublicProfileRequired)
	}
}

func TestCampaignRouteBuilders(t *testing.T) {
	t.Parallel()

	if got := AppCampaign("camp-1"); got != "/app/campaigns/camp-1" {
		t.Fatalf("AppCampaign() = %q", got)
	}
	if got := AppCampaignSessions("camp-1"); got != "/app/campaigns/camp-1/sessions" {
		t.Fatalf("AppCampaignSessions() = %q", got)
	}
	if got := AppCampaignSessionStart("camp-1"); got != "/app/campaigns/camp-1/sessions/start" {
		t.Fatalf("AppCampaignSessionStart() = %q", got)
	}
	if got := AppCampaignSessionEnd("camp-1"); got != "/app/campaigns/camp-1/sessions/end" {
		t.Fatalf("AppCampaignSessionEnd() = %q", got)
	}
	if got := AppCampaignSession("camp-1", "sess-1"); got != "/app/campaigns/camp-1/sessions/sess-1" {
		t.Fatalf("AppCampaignSession() = %q", got)
	}
	if got := AppCampaignParticipants("camp-1"); got != "/app/campaigns/camp-1/participants" {
		t.Fatalf("AppCampaignParticipants() = %q", got)
	}
	if got := AppCampaignParticipantUpdate("camp-1"); got != "/app/campaigns/camp-1/participants/update" {
		t.Fatalf("AppCampaignParticipantUpdate() = %q", got)
	}
	if got := AppCampaignCharacters("camp-1"); got != "/app/campaigns/camp-1/characters" {
		t.Fatalf("AppCampaignCharacters() = %q", got)
	}
	if got := AppCampaignGame("camp-1"); got != "/app/campaigns/camp-1/game" {
		t.Fatalf("AppCampaignGame() = %q", got)
	}
	if got := AppCampaignCharacter("camp-1", "char-1"); got != "/app/campaigns/camp-1/characters/char-1" {
		t.Fatalf("AppCampaignCharacter() = %q", got)
	}
	if got := AppCampaignCharacterCreate("camp-1"); got != "/app/campaigns/camp-1/characters/create" {
		t.Fatalf("AppCampaignCharacterCreate() = %q", got)
	}
	if got := AppCampaignCharacterUpdate("camp-1"); got != "/app/campaigns/camp-1/characters/update" {
		t.Fatalf("AppCampaignCharacterUpdate() = %q", got)
	}
	if got := AppCampaignCharacterControl("camp-1"); got != "/app/campaigns/camp-1/characters/control" {
		t.Fatalf("AppCampaignCharacterControl() = %q", got)
	}
	if got := AppCampaignInvites("camp-1"); got != "/app/campaigns/camp-1/invites" {
		t.Fatalf("AppCampaignInvites() = %q", got)
	}
	if got := AppCampaignInviteCreate("camp-1"); got != "/app/campaigns/camp-1/invites/create" {
		t.Fatalf("AppCampaignInviteCreate() = %q", got)
	}
	if got := AppCampaignInviteRevoke("camp-1"); got != "/app/campaigns/camp-1/invites/revoke" {
		t.Fatalf("AppCampaignInviteRevoke() = %q", got)
	}
}

func TestServeMuxPatternConstants(t *testing.T) {
	t.Parallel()

	if AppCampaignPattern != "/app/campaigns/{campaignID}" {
		t.Fatalf("AppCampaignPattern = %q", AppCampaignPattern)
	}
	if AppCampaignSessionsPattern != "/app/campaigns/{campaignID}/sessions" {
		t.Fatalf("AppCampaignSessionsPattern = %q", AppCampaignSessionsPattern)
	}
	if AppCampaignSessionPattern != "/app/campaigns/{campaignID}/sessions/{sessionID}" {
		t.Fatalf("AppCampaignSessionPattern = %q", AppCampaignSessionPattern)
	}
	if AppCampaignParticipantsPattern != "/app/campaigns/{campaignID}/participants" {
		t.Fatalf("AppCampaignParticipantsPattern = %q", AppCampaignParticipantsPattern)
	}
	if AppCampaignCharactersPattern != "/app/campaigns/{campaignID}/characters" {
		t.Fatalf("AppCampaignCharactersPattern = %q", AppCampaignCharactersPattern)
	}
	if AppCampaignCharacterPattern != "/app/campaigns/{campaignID}/characters/{characterID}" {
		t.Fatalf("AppCampaignCharacterPattern = %q", AppCampaignCharacterPattern)
	}
	if AppCampaignInvitesPattern != "/app/campaigns/{campaignID}/invites" {
		t.Fatalf("AppCampaignInvitesPattern = %q", AppCampaignInvitesPattern)
	}
	if UserProfilePattern != "/u/{username}" {
		t.Fatalf("UserProfilePattern = %q", UserProfilePattern)
	}
	if UserProfilePatternWithTrailingSlash != "/u/{username}/" {
		t.Fatalf("UserProfilePatternWithTrailingSlash = %q", UserProfilePatternWithTrailingSlash)
	}
	if UserProfileRestPattern != "/u/{username}/{rest...}" {
		t.Fatalf("UserProfileRestPattern = %q", UserProfileRestPattern)
	}
	if AppSettingsAIKeyRevokePattern != "/app/settings/ai-keys/{credentialID}/revoke" {
		t.Fatalf("AppSettingsAIKeyRevokePattern = %q", AppSettingsAIKeyRevokePattern)
	}
	if AppNotificationPattern != "/app/notifications/{notificationID}" {
		t.Fatalf("AppNotificationPattern = %q", AppNotificationPattern)
	}
	if AppNotificationOpenPattern != "/app/notifications/{notificationID}/open" {
		t.Fatalf("AppNotificationOpenPattern = %q", AppNotificationOpenPattern)
	}
}

func TestNotificationAndSettingsRouteBuilders(t *testing.T) {
	t.Parallel()

	if got := AppNotificationsOpen("n1"); got != "/app/notifications/n1" {
		t.Fatalf("AppNotificationsOpen() = %q", got)
	}
	if got := AppNotification("n1"); got != "/app/notifications/n1" {
		t.Fatalf("AppNotification() = %q", got)
	}
	if got := AppNotificationOpen("n1"); got != "/app/notifications/n1/open" {
		t.Fatalf("AppNotificationOpen() = %q", got)
	}
	if got := UserProfile("louis"); got != "/u/louis" {
		t.Fatalf("UserProfile() = %q", got)
	}
	if AppSettingsProfile != "/app/settings/profile" {
		t.Fatalf("AppSettingsProfile = %q", AppSettingsProfile)
	}
	if AppSettingsLocale != "/app/settings/locale" {
		t.Fatalf("AppSettingsLocale = %q", AppSettingsLocale)
	}
	if got := AppSettingsProfileWithNotice(SettingsNoticePublicProfileRequired); got != "/app/settings/profile?notice=public-profile-required" {
		t.Fatalf("AppSettingsProfileWithNotice() = %q", got)
	}
	if got := AppSettingsProfileWithNotice("   "); got != "/app/settings/profile" {
		t.Fatalf("AppSettingsProfileWithNotice() empty = %q", got)
	}
	if got := AppSettingsAIKeyRevoke("cred-1"); got != "/app/settings/ai-keys/cred-1/revoke" {
		t.Fatalf("AppSettingsAIKeyRevoke() = %q", got)
	}
}

func TestRouteBuildersEscapeSegments(t *testing.T) {
	t.Parallel()

	if got := AppCampaign("camp/1"); got != "/app/campaigns/camp%2F1" {
		t.Fatalf("AppCampaign() escaped = %q", got)
	}
	if got := AppCampaignSession("camp-1", "sess/1"); got != "/app/campaigns/camp-1/sessions/sess%2F1" {
		t.Fatalf("AppCampaignSession() escaped = %q", got)
	}
	if got := AppCampaignGame("camp/1"); got != "/app/campaigns/camp%2F1/game" {
		t.Fatalf("AppCampaignGame() escaped = %q", got)
	}
	if got := AppCampaignCharacter("camp-1", "char/1"); got != "/app/campaigns/camp-1/characters/char%2F1" {
		t.Fatalf("AppCampaignCharacter() escaped = %q", got)
	}
	if got := AppSettingsAIKeyRevoke("cred/1"); got != "/app/settings/ai-keys/cred%2F1/revoke" {
		t.Fatalf("AppSettingsAIKeyRevoke() escaped = %q", got)
	}
	if got := AppNotificationsOpen("note/1"); got != "/app/notifications/note%2F1" {
		t.Fatalf("AppNotificationsOpen() escaped = %q", got)
	}
	if got := AppNotificationOpen("note/1"); got != "/app/notifications/note%2F1/open" {
		t.Fatalf("AppNotificationOpen() escaped = %q", got)
	}
	if got := UserProfile("name/with/slash"); got != "/u/name%2Fwith%2Fslash" {
		t.Fatalf("UserProfile() escaped = %q", got)
	}
	if got := AppSettingsProfileWithNotice("profile/needed"); got != "/app/settings/profile?notice=profile%2Fneeded" {
		t.Fatalf("AppSettingsProfileWithNotice() escaped = %q", got)
	}
}

func TestEscapeSegmentTrimsWhitespace(t *testing.T) {
	t.Parallel()

	if got := escapeSegment("  camp-1  "); got != "camp-1" {
		t.Fatalf("escapeSegment() = %q", got)
	}
	if got := escapeSegment("  "); got != "" {
		t.Fatalf("escapeSegment() empty = %q", got)
	}
}
