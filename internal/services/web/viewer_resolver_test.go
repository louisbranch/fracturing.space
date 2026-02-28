package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
)

func TestViewerResolverAnonymousReturnsZeroViewer(t *testing.T) {
	t.Parallel()

	r := newViewerResolver(nil, nil, "", func(*http.Request) string { return "" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if v.DisplayName != "" {
		t.Fatalf("DisplayName = %q, want empty", v.DisplayName)
	}
}

func TestViewerResolverNilSocialClientReturnsFallback(t *testing.T) {
	t.Parallel()

	r := newViewerResolver(nil, nil, "https://cdn.example.com", func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if v.DisplayName != "Adventurer" {
		t.Fatalf("DisplayName = %q, want %q", v.DisplayName, "Adventurer")
	}
	if v.ProfileURL != "/app/settings/profile" {
		t.Fatalf("ProfileURL = %q, want %q", v.ProfileURL, "/app/settings/profile")
	}
}

func TestViewerResolverWithSocialClientUsesProfileData(t *testing.T) {
	t.Parallel()

	social := &fakeSocialClient{
		getUserProfileResp: &socialv1.GetUserProfileResponse{
			UserProfile: &socialv1.UserProfile{Username: "alice", Name: "Alice"},
		},
	}
	r := newViewerResolver(social, nil, "https://cdn.example.com", func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if v.DisplayName != "Alice" {
		t.Fatalf("DisplayName = %q, want %q", v.DisplayName, "Alice")
	}
	if v.ProfileURL != "/u/alice" {
		t.Fatalf("ProfileURL = %q, want %q", v.ProfileURL, "/u/alice")
	}
}

func TestViewerResolverUnreadNotifications(t *testing.T) {
	t.Parallel()

	notif := fakeWebNotificationClient{
		unreadResp: &notificationsv1.GetUnreadNotificationStatusResponse{HasUnread: true},
	}
	r := newViewerResolver(nil, notif, "", func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if !v.HasUnreadNotifications {
		t.Fatalf("HasUnreadNotifications = false, want true")
	}
}

func TestViewerResolverNoUnreadNotifications(t *testing.T) {
	t.Parallel()

	notif := fakeWebNotificationClient{
		unreadResp: &notificationsv1.GetUnreadNotificationStatusResponse{HasUnread: false},
	}
	r := newViewerResolver(nil, notif, "", func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if v.HasUnreadNotifications {
		t.Fatalf("HasUnreadNotifications = true, want false")
	}
}
