package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestViewerResolverAnonymousReturnsZeroViewer(t *testing.T) {
	t.Parallel()

	r := newViewerResolver(nil, nil, nil, "", func(*http.Request) string { return "" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if v.DisplayName != "" {
		t.Fatalf("DisplayName = %q, want empty", v.DisplayName)
	}
}

func TestViewerResolverNilResolverReturnsZeroViewer(t *testing.T) {
	t.Parallel()

	r := newViewerResolver(nil, nil, nil, "", nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)
	if v != (module.Viewer{}) {
		t.Fatalf("resolveViewer() = %+v, want zero viewer", v)
	}
}

func TestViewerResolverWhitespaceUserIDReturnsZeroViewer(t *testing.T) {
	t.Parallel()

	r := newViewerResolver(nil, nil, nil, "", func(*http.Request) string { return "   " })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)
	if v != (module.Viewer{}) {
		t.Fatalf("resolveViewer() = %+v, want zero viewer", v)
	}
}

func TestViewerResolverNilRequestReturnsZeroViewer(t *testing.T) {
	t.Parallel()

	r := newViewerResolver(nil, nil, nil, "", func(*http.Request) string { return "user-1" })
	v := r.resolveViewer(nil)
	if v != (module.Viewer{}) {
		t.Fatalf("resolveViewer(nil) = %+v, want zero viewer", v)
	}
}

func TestViewerResolverNilSocialClientReturnsAuthBackedProfileLink(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "alice"}},
	}
	r := newViewerResolver(account, nil, nil, "https://cdn.example.com", func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if v.DisplayName != "Adventurer" {
		t.Fatalf("DisplayName = %q, want %q", v.DisplayName, "Adventurer")
	}
	if v.ProfileURL != "/u/alice" {
		t.Fatalf("ProfileURL = %q, want %q", v.ProfileURL, "/u/alice")
	}
	if !strings.Contains(v.AvatarURL, "/avatar_set/v1/") {
		t.Fatalf("AvatarURL = %q, want people-set path", v.AvatarURL)
	}
}

func TestViewerResolverWithoutAccountProfileDoesNotFallBackToSettingsProfile(t *testing.T) {
	t.Parallel()

	r := newViewerResolver(nil, nil, nil, "https://cdn.example.com", func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if v.ProfileURL != "/app/dashboard" {
		t.Fatalf("ProfileURL = %q, want %q", v.ProfileURL, "/app/dashboard")
	}
}

func TestViewerResolverWithSocialClientUsesProfileData(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "alice"}},
	}
	social := &fakeSocialClient{
		getUserProfileResp: &socialv1.GetUserProfileResponse{
			UserProfile: &socialv1.UserProfile{Name: "Alice"},
		},
	}
	r := newViewerResolver(account, social, nil, "https://cdn.example.com", func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if v.DisplayName != "Alice" {
		t.Fatalf("DisplayName = %q, want %q", v.DisplayName, "Alice")
	}
	if v.ProfileURL != "/u/alice" {
		t.Fatalf("ProfileURL = %q, want %q", v.ProfileURL, "/u/alice")
	}
}

func TestViewerResolverWithSocialClientKeepsAuthBackedProfileRouteWhenSocialRecordHasNoUsername(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "alice"}},
	}
	social := &fakeSocialClient{
		getUserProfileResp: &socialv1.GetUserProfileResponse{
			UserProfile: &socialv1.UserProfile{Name: "Alice"},
		},
	}
	r := newViewerResolver(account, social, nil, "https://cdn.example.com", func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if v.ProfileURL != "/u/alice" {
		t.Fatalf("ProfileURL = %q, want %q", v.ProfileURL, "/u/alice")
	}
}

func TestViewerResolverWithSocialClientNotFoundKeepsAuthBackedProfileRoute(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "alice"}},
	}
	social := &fakeSocialClient{
		getUserProfileErr: status.Error(codes.NotFound, "profile not found"),
	}
	r := newViewerResolver(account, social, nil, "https://cdn.example.com", func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if v.ProfileURL != "/u/alice" {
		t.Fatalf("ProfileURL = %q, want %q", v.ProfileURL, "/u/alice")
	}
	if !strings.Contains(v.AvatarURL, "/avatar_set/v1/") {
		t.Fatalf("AvatarURL = %q, want people-set path", v.AvatarURL)
	}
}

func TestViewerResolverUnreadNotifications(t *testing.T) {
	t.Parallel()

	notif := fakeWebNotificationClient{
		unreadResp: &notificationsv1.GetUnreadNotificationStatusResponse{HasUnread: true},
	}
	r := newViewerResolver(nil, nil, notif, "", func(*http.Request) string { return "user-1" })
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
	r := newViewerResolver(nil, nil, notif, "", func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.resolveViewer(req)

	if v.HasUnreadNotifications {
		t.Fatalf("HasUnreadNotifications = true, want false")
	}
}
