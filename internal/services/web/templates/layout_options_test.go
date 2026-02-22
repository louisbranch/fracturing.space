package templates

import (
	"testing"

	"golang.org/x/text/message"
)

type localizedTitle struct{}

func (localizedTitle) Sprintf(key message.Reference, _ ...any) string {
	if text, ok := key.(string); ok {
		return "localized:" + text
	}
	return ""
}

func TestLayoutOptionsForPageBuildsCommonValues(t *testing.T) {
	page := PageContext{
		Lang:                   "en-US",
		AppName:                "app-name",
		Loc:                    localizedTitle{},
		CurrentPath:            "/profile",
		CampaignName:           "Campaign",
		UserName:               "Alice",
		UserAvatarURL:          "https://example.com/avatar.png",
		HasUnreadNotifications: true,
	}
	got := LayoutOptionsForPage(page, "layout.profile", true)

	if got.Title != "localized:layout.profile" {
		t.Fatalf("Title = %q, want %q", got.Title, "localized:layout.profile")
	}
	if got.Lang != page.Lang {
		t.Fatalf("Lang = %q, want %q", got.Lang, page.Lang)
	}
	if got.CurrentPath != page.CurrentPath {
		t.Fatalf("CurrentPath = %q, want %q", got.CurrentPath, page.CurrentPath)
	}
	if got.CampaignName != page.CampaignName {
		t.Fatalf("CampaignName = %q, want %q", got.CampaignName, page.CampaignName)
	}
	if got.UserName != page.UserName {
		t.Fatalf("UserName = %q, want %q", got.UserName, page.UserName)
	}
	if got.UserAvatarURL != page.UserAvatarURL {
		t.Fatalf("UserAvatarURL = %q, want %q", got.UserAvatarURL, page.UserAvatarURL)
	}
	if got.HasUnreadNotifications != page.HasUnreadNotifications {
		t.Fatalf("HasUnreadNotifications = %t, want %t", got.HasUnreadNotifications, page.HasUnreadNotifications)
	}
	if got.AppName != page.AppName {
		t.Fatalf("AppName = %q, want %q", got.AppName, page.AppName)
	}
	if !got.UseCustomBreadcrumbs {
		t.Fatal("UseCustomBreadcrumbs = false, want true")
	}
}
