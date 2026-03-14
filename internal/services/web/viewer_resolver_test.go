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
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestViewerResolverAnonymousReturnsZeroViewer(t *testing.T) {
	t.Parallel()

	r := principal.New(principal.Dependencies{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.ResolveViewer(req)

	if v.DisplayName != "" {
		t.Fatalf("DisplayName = %q, want empty", v.DisplayName)
	}
}

func TestViewerResolverNilResolverReturnsZeroViewer(t *testing.T) {
	t.Parallel()

	r := principal.New(principal.Dependencies{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	v := r.ResolveViewer(req)
	if v != (module.Viewer{}) {
		t.Fatalf("ResolveViewer() = %+v, want zero viewer", v)
	}
}

func TestViewerResolverUnknownSessionReturnsZeroViewer(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{SessionClient: auth})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "unknown-session"})
	v := r.ResolveViewer(req)
	if v != (module.Viewer{}) {
		t.Fatalf("ResolveViewer() = %+v, want zero viewer", v)
	}
}

func TestViewerResolverNilRequestReturnsZeroViewer(t *testing.T) {
	t.Parallel()

	r := principal.New(principal.Dependencies{})
	v := r.ResolveViewer(nil)
	if v != (module.Viewer{}) {
		t.Fatalf("ResolveViewer(nil) = %+v, want zero viewer", v)
	}
}

func TestViewerResolverNilSocialClientReturnsAuthBackedProfileLink(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "alice"}},
	}
	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{
		SessionClient: auth,
		AccountClient: account,
		AssetBaseURL:  "https://cdn.example.com",
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	v := r.ResolveViewer(req)

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

	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{
		SessionClient: auth,
		AssetBaseURL:  "https://cdn.example.com",
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	v := r.ResolveViewer(req)

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
	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{
		SessionClient: auth,
		AccountClient: account,
		SocialClient:  social,
		AssetBaseURL:  "https://cdn.example.com",
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	v := r.ResolveViewer(req)

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
	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{
		SessionClient: auth,
		AccountClient: account,
		SocialClient:  social,
		AssetBaseURL:  "https://cdn.example.com",
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	v := r.ResolveViewer(req)

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
	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{
		SessionClient: auth,
		AccountClient: account,
		SocialClient:  social,
		AssetBaseURL:  "https://cdn.example.com",
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	v := r.ResolveViewer(req)

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
	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{
		SessionClient:      auth,
		NotificationClient: notif,
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	v := r.ResolveViewer(req)

	if !v.NotificationsAvailable {
		t.Fatalf("NotificationsAvailable = false, want true")
	}
	if !v.HasUnreadNotifications {
		t.Fatalf("HasUnreadNotifications = false, want true")
	}
}

func TestViewerResolverNoUnreadNotifications(t *testing.T) {
	t.Parallel()

	notif := fakeWebNotificationClient{
		unreadResp: &notificationsv1.GetUnreadNotificationStatusResponse{HasUnread: false},
	}
	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{
		SessionClient:      auth,
		NotificationClient: notif,
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	v := r.ResolveViewer(req)

	if !v.NotificationsAvailable {
		t.Fatalf("NotificationsAvailable = false, want true")
	}
	if v.HasUnreadNotifications {
		t.Fatalf("HasUnreadNotifications = true, want false")
	}
}

func TestViewerResolverWithoutNotificationClientOmitsNotificationsAvailability(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{SessionClient: auth})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	v := r.ResolveViewer(req)

	if v.NotificationsAvailable {
		t.Fatalf("NotificationsAvailable = true, want false")
	}
	if v.HasUnreadNotifications {
		t.Fatalf("HasUnreadNotifications = true, want false")
	}
}
