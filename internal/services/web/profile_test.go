package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestAppProfileRouteRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppProfileRouteRejectsUnsupportedMethod(t *testing.T) {
	h := &handler{
		config:        Config{AuthBaseURL: "http://auth.local"},
		sessions:      newSessionStore(),
		pendingFlows:  newPendingFlowStore(),
		accountClient: &fakeAccountClient{},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true

	req := httptest.NewRequest(http.MethodDelete, "/profile", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppProfile(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
	if allow := w.Header().Get("Allow"); allow != http.MethodGet+", "+http.MethodPost {
		t.Fatalf("Allow = %q, want %q", allow, http.MethodGet+", "+http.MethodPost)
	}
}

func TestAppProfileRouteRendersFormForAuthenticatedUser(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
		accountClient: &fakeAccountClient{
			getProfileResp: &authv1.GetProfileResponse{
				Profile: &authv1.AccountProfile{
					UserId: "user-1",
					Name:   "Alice Profile",
					Locale: commonv1.Locale_LOCALE_PT_BR,
				},
			},
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppProfile(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, `value="Alice Profile"`) {
		t.Fatalf("body should include pre-filled name input, got %q", body)
	}
	if !strings.Contains(body, `<option value="pt-BR" selected>`) {
		t.Fatalf("body should include selected locale option, got %q", body)
	}
	if !strings.Contains(body, `action="/profile"`) {
		t.Fatalf("body should include profile form action, got %q", body)
	}
	if !strings.Contains(body, `type="submit"`) {
		t.Fatalf("body should include submit button, got %q", body)
	}
}

func TestAppProfileRouteRendersFormWhenProfileDoesNotExist(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
		accountClient: &fakeAccountClient{
			getProfileErr: status.Error(codes.NotFound, "profile not found"),
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppProfile(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, `value="Alice"`) {
		t.Fatalf("body should include session user name fallback, got %q", body)
	}
}

func TestAppProfileRouteUpdatesProfileOnPost(t *testing.T) {
	fakeAccount := &fakeAccountClient{}
	h := &handler{
		config:        Config{AuthBaseURL: "http://auth.local"},
		sessions:      newSessionStore(),
		pendingFlows:  newPendingFlowStore(),
		accountClient: fakeAccount,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true

	body := url.Values{}
	body.Set("name", "Nova Name")
	body.Set("locale", "pt-BR")
	req := httptest.NewRequest(http.MethodPost, "/profile", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppProfile(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/profile" {
		t.Fatalf("Location = %q, want /profile", location)
	}
	if fakeAccount.lastUpdateReq == nil {
		t.Fatal("expected UpdateProfile call")
	}
	if got := fakeAccount.lastUpdateReq.GetUserId(); strings.TrimSpace(got) != "user-1" {
		t.Fatalf("user_id = %q, want user-1", got)
	}
	if got := fakeAccount.lastUpdateReq.GetName(); strings.TrimSpace(got) != "Nova Name" {
		t.Fatalf("name = %q, want Nova Name", got)
	}
	if got := fakeAccount.lastUpdateReq.GetLocale(); got != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("locale = %v, want %v", got, commonv1.Locale_LOCALE_PT_BR)
	}
}

type fakeAccountClient struct {
	getProfileResp *authv1.GetProfileResponse
	getProfileErr  error

	updateProfileResp *authv1.UpdateProfileResponse
	updateProfileErr  error
	lastUpdateReq     *authv1.UpdateProfileRequest
}

func (f *fakeAccountClient) GetProfile(ctx context.Context, in *authv1.GetProfileRequest, opts ...grpc.CallOption) (*authv1.GetProfileResponse, error) {
	if f.getProfileErr != nil {
		return nil, f.getProfileErr
	}
	if f.getProfileResp != nil {
		return f.getProfileResp, nil
	}
	return &authv1.GetProfileResponse{}, nil
}

func (f *fakeAccountClient) UpdateProfile(ctx context.Context, in *authv1.UpdateProfileRequest, opts ...grpc.CallOption) (*authv1.UpdateProfileResponse, error) {
	f.lastUpdateReq = in
	if f.updateProfileErr != nil {
		return nil, f.updateProfileErr
	}
	if f.updateProfileResp != nil {
		return f.updateProfileResp, nil
	}
	return &authv1.UpdateProfileResponse{}, nil
}
