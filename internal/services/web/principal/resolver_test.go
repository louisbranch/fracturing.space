package principal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"google.golang.org/grpc"
)

func TestResolverResolveUserIDAndSignedIn(t *testing.T) {
	t.Parallel()

	session := &fakeSessionClient{sessions: map[string]string{"ws-1": "user-1"}}
	resolver := New(Dependencies{SessionClient: session})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})

	if got := resolver.ResolveUserID(request); got != "user-1" {
		t.Fatalf("ResolveUserID() = %q, want %q", got, "user-1")
	}
	if !resolver.ResolveSignedIn(request) {
		t.Fatalf("ResolveSignedIn() = false, want true")
	}
}

func TestResolverAuthRequiredRejectsUnknownSession(t *testing.T) {
	t.Parallel()

	resolver := New(Dependencies{SessionClient: &fakeSessionClient{sessions: map[string]string{}}})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: "web_session", Value: "unknown"})

	if resolver.AuthRequired(request) {
		t.Fatalf("AuthRequired() = true, want false for unknown session")
	}
}

func TestResolverResolveLanguageUsesAccountLocale(t *testing.T) {
	t.Parallel()

	session := &fakeSessionClient{sessions: map[string]string{"ws-1": "user-1"}}
	account := &fakeAccountClient{profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR}}
	resolver := New(Dependencies{
		SessionClient: session,
		AccountClient: account,
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})

	if got := resolver.ResolveLanguage(request); got != "pt-BR" {
		t.Fatalf("ResolveLanguage() = %q, want %q", got, "pt-BR")
	}
}

func TestResolverResolveViewerUsesAccountSocialAndUnreadState(t *testing.T) {
	t.Parallel()

	session := &fakeSessionClient{sessions: map[string]string{"ws-1": "user-1"}}
	account := &fakeAccountClient{profile: &authv1.AccountProfile{Username: "alice"}}
	social := &fakeSocialClient{profile: &socialv1.UserProfile{Name: "Alice"}}
	notifications := &fakeNotificationClient{
		resp: &notificationsv1.GetUnreadNotificationStatusResponse{HasUnread: true},
	}
	resolver := New(Dependencies{
		SessionClient:      session,
		AccountClient:      account,
		SocialClient:       social,
		NotificationClient: notifications,
		AssetBaseURL:       "https://cdn.example.com",
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})

	viewer := resolver.ResolveViewer(request)
	if viewer.DisplayName != "Alice" {
		t.Fatalf("DisplayName = %q, want %q", viewer.DisplayName, "Alice")
	}
	if viewer.ProfileURL != "/u/alice" {
		t.Fatalf("ProfileURL = %q, want %q", viewer.ProfileURL, "/u/alice")
	}
	if !viewer.NotificationsAvailable {
		t.Fatalf("NotificationsAvailable = false, want true")
	}
	if !viewer.HasUnreadNotifications {
		t.Fatalf("HasUnreadNotifications = false, want true")
	}
}

func TestResolverMiddlewareSharesSessionAndAccountLookups(t *testing.T) {
	t.Parallel()

	session := &fakeSessionClient{sessions: map[string]string{"ws-1": "user-1"}}
	account := &fakeAccountClient{
		profile: &authv1.AccountProfile{
			Username: "alice",
			Locale:   commonv1.Locale_LOCALE_PT_BR,
		},
	}
	resolver := New(Dependencies{
		SessionClient: session,
		AccountClient: account,
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: "web_session", Value: "ws-1"})

	handler := resolver.Middleware()(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		_ = resolver.ResolveUserID(request)
		_ = resolver.ResolveSignedIn(request)
		_ = resolver.ResolveViewer(request)
		_ = resolver.ResolveLanguage(request)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if session.calls != 1 {
		t.Fatalf("GetWebSession calls = %d, want %d", session.calls, 1)
	}
	if account.calls != 1 {
		t.Fatalf("GetProfile calls = %d, want %d", account.calls, 1)
	}
}

type fakeSessionClient struct {
	sessions map[string]string
	calls    int
}

func (f *fakeSessionClient) GetWebSession(_ context.Context, req *authv1.GetWebSessionRequest, _ ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	f.calls++
	userID, ok := f.sessions[req.GetSessionId()]
	if !ok {
		return nil, errors.New("session not found")
	}
	return &authv1.GetWebSessionResponse{
		Session: &authv1.WebSession{Id: req.GetSessionId(), UserId: userID},
	}, nil
}

type fakeAccountClient struct {
	profile *authv1.AccountProfile
	calls   int
}

func (f *fakeAccountClient) GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error) {
	f.calls++
	return &authv1.GetProfileResponse{Profile: f.profile}, nil
}

type fakeNotificationClient struct {
	resp *notificationsv1.GetUnreadNotificationStatusResponse
}

func (f *fakeNotificationClient) GetUnreadNotificationStatus(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest, ...grpc.CallOption) (*notificationsv1.GetUnreadNotificationStatusResponse, error) {
	return f.resp, nil
}

type fakeSocialClient struct {
	profile *socialv1.UserProfile
}

func (f *fakeSocialClient) GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	return &socialv1.GetUserProfileResponse{UserProfile: f.profile}, nil
}
