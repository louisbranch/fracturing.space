package routepath

import "testing"

func TestCampaignRouteBuilders(t *testing.T) {
	t.Parallel()

	if got := Campaign("camp-1"); got != "/app/campaigns/camp-1" {
		t.Fatalf("Campaign(%q) = %q", "camp-1", got)
	}
	if got := CampaignSessions("camp-1"); got != "/app/campaigns/camp-1/sessions" {
		t.Fatalf("CampaignSessions(%q) = %q", "camp-1", got)
	}
	if got := CampaignParticipants("camp-1"); got != "/app/campaigns/camp-1/participants" {
		t.Fatalf("CampaignParticipants(%q) = %q", "camp-1", got)
	}
	if got := CampaignCharacters("camp-1"); got != "/app/campaigns/camp-1/characters" {
		t.Fatalf("CampaignCharacters(%q) = %q", "camp-1", got)
	}
	if got := CampaignInvites("camp-1"); got != "/app/campaigns/camp-1/invites" {
		t.Fatalf("CampaignInvites(%q) = %q", "camp-1", got)
	}
}

func TestCampaignRouteBuildersEscapeIDs(t *testing.T) {
	t.Parallel()

	if got := Campaign("camp/1"); got != "/app/campaigns/camp%2F1" {
		t.Fatalf("Campaign(%q) = %q", "camp/1", got)
	}
	if got := CampaignSession("camp-1", "sess/1"); got != "/app/campaigns/camp-1/sessions/sess%2F1" {
		t.Fatalf("CampaignSession(%q, %q) = %q", "camp-1", "sess/1", got)
	}
	if got := CampaignCharacter("camp-1", "char/1"); got != "/app/campaigns/camp-1/characters/char%2F1" {
		t.Fatalf("CampaignCharacter(%q, %q) = %q", "camp-1", "char/1", got)
	}
}

func TestTopLevelRouteConstants(t *testing.T) {
	t.Parallel()

	if AppRoot != "/app" {
		t.Fatalf("AppRoot = %q", AppRoot)
	}
	if AppCampaigns != "/app/campaigns" {
		t.Fatalf("AppCampaigns = %q", AppCampaigns)
	}
	if AppProfile != "/app/profile" {
		t.Fatalf("AppProfile = %q", AppProfile)
	}
	if AppSettings != "/app/settings" {
		t.Fatalf("AppSettings = %q", AppSettings)
	}
	if AppInvites != "/app/invites" {
		t.Fatalf("AppInvites = %q", AppInvites)
	}
	if AppNotifications != "/app/notifications" {
		t.Fatalf("AppNotifications = %q", AppNotifications)
	}
}

func TestPublicRouteConstants(t *testing.T) {
	t.Parallel()

	if UserProfilePrefix != "/u/" {
		t.Fatalf("UserProfilePrefix = %q", UserProfilePrefix)
	}
	if Discover != "/discover" {
		t.Fatalf("Discover = %q", Discover)
	}
	if DiscoverCampaigns != "/discover/campaigns" {
		t.Fatalf("DiscoverCampaigns = %q", DiscoverCampaigns)
	}
	if DiscoverCampaignsPrefix != "/discover/campaigns/" {
		t.Fatalf("DiscoverCampaignsPrefix = %q", DiscoverCampaignsPrefix)
	}
}

func TestPublicRouteBuilders(t *testing.T) {
	t.Parallel()

	if got := UserProfile("alice"); got != "/u/alice" {
		t.Fatalf("UserProfile(%q) = %q", "alice", got)
	}
	if got := DiscoverCampaign("camp-1"); got != "/discover/campaigns/camp-1" {
		t.Fatalf("DiscoverCampaign(%q) = %q", "camp-1", got)
	}
	if got := DiscoverCampaign("camp/1"); got != "/discover/campaigns/camp%2F1" {
		t.Fatalf("DiscoverCampaign(%q) = %q", "camp/1", got)
	}
}
