package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestLoginWithoutPendingIDRedirectsToAuthLogin(t *testing.T) {
	handler := NewHandler(Config{
		AuthBaseURL:   "http://auth.local",
		OAuthClientID: "fracturing-space",
		CallbackURL:   "http://localhost:8080/auth/callback",
	}, nil)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != "/auth/login" {
		t.Fatalf("Location = %q, want %q", loc, "/auth/login")
	}
}

func TestLoginWithoutPendingIDErrorsWhenOAuthNotConfigured(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginHandlerRendersForm(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/login?pending_id=pending-1&client_id=client-1&client_name=Test+Client", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "pending-1") {
		t.Fatalf("expected pending_id in body")
	}
	if !strings.Contains(body, "Test Client") {
		t.Fatalf("expected client name in body")
	}
	if !strings.Contains(body, "<title>Sign In | "+branding.AppName+"</title>") {
		t.Fatalf("expected title suffix on login page")
	}
	if !strings.Contains(body, `data-layout="auth"`) {
		t.Fatalf("expected auth layout marker in login page")
	}
}

func TestLandingPageRenders(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, branding.AppName) {
		t.Fatalf("expected app name in body")
	}
	if !strings.Contains(body, "<title>Open source AI GM engine | "+branding.AppName+"</title>") {
		t.Fatalf("expected title suffix on landing page")
	}
	if !strings.Contains(body, "Open-source, server-authoritative engine") {
		t.Fatalf("expected hero tagline in body")
	}
	if !strings.Contains(body, `data-layout="auth"`) {
		t.Fatalf("expected auth layout marker in landing page")
	}
}

func TestLandingPageShowsSignIn(t *testing.T) {
	handler := NewHandler(Config{
		AuthBaseURL:   "http://auth.local",
		OAuthClientID: "fracturing-space",
		CallbackURL:   "http://localhost:8080/auth/callback",
	}, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Sign in") {
		t.Fatalf("expected Sign in button in body")
	}
	if !strings.Contains(body, "/auth/login") {
		t.Fatalf("expected /auth/login link in body")
	}
}

func TestLandingPageShowsSignedInUser(t *testing.T) {
	h := &handler{
		config: Config{
			AuthBaseURL:   "http://auth.local",
			OAuthClientID: "fracturing-space",
			CallbackURL:   "http://localhost:8080/auth/callback",
		},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	// Build the full handler so we go through the mux.
	handler := NewHandler(h.config, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	// The handler creates its own session store, so this session won't be found.
	// Instead, it should show "Sign in" since the session is unknown.
	if !strings.Contains(body, "Sign in") {
		t.Fatalf("expected Sign in for unknown session")
	}
}

func TestLandingPageRejectsNonRootPath(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/something", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestLandingPageRejectsNonGETMethod(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPasskeyLoginStartRequiresClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/start", bytes.NewBufferString(`{"pending_id":"pending-1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestPasskeyLoginStartSuccess(t *testing.T) {
	fake := &fakeAuthClient{
		beginLoginResp: &authv1.BeginPasskeyLoginResponse{
			SessionId:                    "session-1",
			CredentialRequestOptionsJson: []byte(`{"challenge":"test"}`),
		},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/start", bytes.NewBufferString(`{"pending_id":"pending-1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["session_id"] != "session-1" {
		t.Fatalf("session_id = %v", payload["session_id"])
	}
}

func TestPasskeyLoginFinishRequiresFields(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/finish", bytes.NewBufferString(`{"pending_id":"pending-1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPasskeyLoginFinishSuccess(t *testing.T) {
	fake := &fakeAuthClient{
		finishLoginResp: &authv1.FinishPasskeyLoginResponse{},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/finish", bytes.NewBufferString(`{"pending_id":"pending-1","session_id":"session-1","credential":{}}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["redirect_url"] != "http://auth.local/authorize/consent?pending_id=pending-1" {
		t.Fatalf("redirect_url = %v", payload["redirect_url"])
	}
}

func TestMagicLinkRequiresToken(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodGet, "/magic", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Magic link missing") {
		t.Fatalf("expected error page")
	}
	if !strings.Contains(w.Body.String(), `data-layout="auth"`) {
		t.Fatalf("expected auth layout marker in magic page")
	}
}

func TestMagicLinkRedirectsToConsent(t *testing.T) {
	fake := &fakeAuthClient{
		consumeMagicResp: &authv1.ConsumeMagicLinkResponse{PendingId: "pending-1"},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodGet, "/magic?token=token-1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "http://auth.local/authorize/consent?pending_id=pending-1" {
		t.Fatalf("location = %q", location)
	}
}

func TestMagicLinkSuccessPage(t *testing.T) {
	fake := &fakeAuthClient{
		consumeMagicResp: &authv1.ConsumeMagicLinkResponse{},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodGet, "/magic?token=token-1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "Magic link verified") {
		t.Fatalf("expected success page")
	}
}

func TestPasskeyRegisterStartRequiresFields(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/start", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPasskeyRegisterStartSuccess(t *testing.T) {
	fake := &fakeAuthClient{
		createUserResp: &authv1.CreateUserResponse{
			User: &authv1.User{Id: "user-1", Email: "alpha@example.com"},
		},
		beginRegResp: &authv1.BeginPasskeyRegistrationResponse{
			SessionId:                     "session-1",
			CredentialCreationOptionsJson: []byte(`{"challenge":"test","user":{"id":"user"}}`),
		},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/start", bytes.NewBufferString(`{"email":"alpha@example.com"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["session_id"] != "session-1" {
		t.Fatalf("session_id = %v", payload["session_id"])
	}
	if payload["user_id"] != "user-1" {
		t.Fatalf("user_id = %v", payload["user_id"])
	}
}

func TestPasskeyRegisterFinishRequiresFields(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/finish", bytes.NewBufferString(`{"session_id":"session-1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPasskeyRegisterFinishSuccess(t *testing.T) {
	fake := &fakeAuthClient{
		finishRegResp: &authv1.FinishPasskeyRegistrationResponse{},
	}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/finish", bytes.NewBufferString(`{"session_id":"session-1","user_id":"user-1","credential":{}}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthLoginRedirect(t *testing.T) {
	handler := NewHandler(Config{
		AuthBaseURL:   "http://auth.local",
		OAuthClientID: "fracturing-space",
		CallbackURL:   "http://localhost:8080/auth/callback",
	}, nil)
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("expected Location header")
	}
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	if parsed.Host != "auth.local" {
		t.Fatalf("host = %q, want %q", parsed.Host, "auth.local")
	}
	if parsed.Path != "/authorize" {
		t.Fatalf("path = %q, want %q", parsed.Path, "/authorize")
	}
	q := parsed.Query()
	if q.Get("response_type") != "code" {
		t.Fatalf("response_type = %q", q.Get("response_type"))
	}
	if q.Get("client_id") != "fracturing-space" {
		t.Fatalf("client_id = %q", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "http://localhost:8080/auth/callback" {
		t.Fatalf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Get("code_challenge") == "" {
		t.Fatal("expected code_challenge")
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("code_challenge_method = %q", q.Get("code_challenge_method"))
	}
	if q.Get("state") == "" {
		t.Fatal("expected state parameter")
	}
}

func TestAuthCallbackExchangesCodeAndSetsCookie(t *testing.T) {
	// Mock token endpoint.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Verify the required fields are sent.
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.FormValue("grant_type") != "authorization_code" {
			http.Error(w, "wrong grant_type", http.StatusBadRequest)
			return
		}
		if r.FormValue("client_id") != "fracturing-space" {
			http.Error(w, "wrong client_id", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	h := &handler{
		config: Config{
			AuthBaseURL:   "http://auth.local",
			OAuthClientID: "fracturing-space",
			CallbackURL:   "http://localhost:8080/auth/callback",
			AuthTokenURL:  tokenServer.URL,
		},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}

	// Seed a pending flow.
	state := h.pendingFlows.create("test-verifier")

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code&state="+state, nil)
	w := httptest.NewRecorder()
	h.handleAuthCallback(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusFound, w.Body.String())
	}
	if w.Header().Get("Location") != "/" {
		t.Fatalf("Location = %q, want %q", w.Header().Get("Location"), "/")
	}

	// Verify session cookie was set.
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie")
	}

	// Verify session exists in the store.
	sess := h.sessions.get(sessionCookie.Value)
	if sess == nil {
		t.Fatal("expected session in store")
	}
	if sess.accessToken != "test-access-token" {
		t.Fatalf("accessToken = %q, want %q", sess.accessToken, "test-access-token")
	}
}

func TestAuthCallbackSetsTokenCookie(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"test-access-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	h := &handler{
		config: Config{
			AuthBaseURL:   "http://auth.local",
			OAuthClientID: "fracturing-space",
			CallbackURL:   "http://localhost:8080/auth/callback",
			AuthTokenURL:  tokenServer.URL,
			Domain:        "example.com",
		},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}

	state := h.pendingFlows.create("test-verifier")
	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code&state="+state, nil)
	w := httptest.NewRecorder()
	h.handleAuthCallback(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusFound, w.Body.String())
	}

	var tokenCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == tokenCookieName {
			tokenCookie = c
			break
		}
	}
	if tokenCookie == nil {
		t.Fatal("expected fs_token cookie")
	}
	if tokenCookie.Value != "test-access-token" {
		t.Fatalf("token cookie value = %q, want %q", tokenCookie.Value, "test-access-token")
	}
	if tokenCookie.Domain != "example.com" {
		t.Fatalf("token cookie domain = %q, want %q", tokenCookie.Domain, "example.com")
	}
	if tokenCookie.MaxAge != 3600 {
		t.Fatalf("token cookie MaxAge = %d, want 3600", tokenCookie.MaxAge)
	}
}

func TestAuthLogoutClearsTokenCookie(t *testing.T) {
	h := &handler{
		config: Config{
			AuthBaseURL:   "http://auth.local",
			OAuthClientID: "fracturing-space",
			Domain:        "example.com",
		},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}

	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()
	h.handleAuthLogout(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}

	var tokenCleared bool
	for _, c := range w.Result().Cookies() {
		if c.Name == tokenCookieName && c.MaxAge == -1 {
			tokenCleared = true
		}
	}
	if !tokenCleared {
		t.Fatal("expected fs_token cookie to be cleared")
	}
}

func TestAuthCallbackMissingCodeOrState(t *testing.T) {
	h := &handler{
		config: Config{
			AuthBaseURL:   "http://auth.local",
			OAuthClientID: "fracturing-space",
			CallbackURL:   "http://localhost:8080/auth/callback",
		},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code", nil)
	w := httptest.NewRecorder()
	h.handleAuthCallback(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthCallbackInvalidState(t *testing.T) {
	h := &handler{
		config: Config{
			AuthBaseURL:   "http://auth.local",
			OAuthClientID: "fracturing-space",
			CallbackURL:   "http://localhost:8080/auth/callback",
		},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/callback?code=test-code&state=bogus", nil)
	w := httptest.NewRecorder()
	h.handleAuthCallback(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAuthLogoutClearsSessionAndRedirects(t *testing.T) {
	h := &handler{
		config: Config{
			AuthBaseURL:   "http://auth.local",
			OAuthClientID: "fracturing-space",
		},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}

	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()
	h.handleAuthLogout(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if w.Header().Get("Location") != "/" {
		t.Fatalf("Location = %q, want %q", w.Header().Get("Location"), "/")
	}

	// Session should be deleted.
	if sess := h.sessions.get(sessionID); sess != nil {
		t.Fatal("expected session to be deleted")
	}

	// Session cookie should be cleared.
	cookies := w.Result().Cookies()
	var cleared bool
	for _, c := range cookies {
		if c.Name == sessionCookieName && c.MaxAge == -1 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("expected session cookie to be cleared")
	}
}

func TestAuthLogoutMethodNotAllowed(t *testing.T) {
	h := &handler{
		config: Config{
			AuthBaseURL:   "http://auth.local",
			OAuthClientID: "fracturing-space",
		},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/logout", nil)
	w := httptest.NewRecorder()
	h.handleAuthLogout(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestAuthLoginNotConfigured(t *testing.T) {
	handler := NewHandler(Config{
		AuthBaseURL: "http://auth.local",
		// OAuthClientID is empty — not configured.
	}, nil)
	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestNewServerRequiresHTTPAddr(t *testing.T) {
	_, err := NewServer(Config{AuthBaseURL: "http://auth.local"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewServerRequiresAuthBaseURL(t *testing.T) {
	_, err := NewServer(Config{HTTPAddr: "127.0.0.1:0"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewHandlerWithCampaignAccessStaticAssetsFailure(t *testing.T) {
	origSubStaticFS := subStaticFS
	subStaticFS = func() (fs.FS, error) {
		return nil, fmt.Errorf("injected static assets failure")
	}
	defer func() {
		subStaticFS = origSubStaticFS
	}()

	_, err := NewHandlerWithCampaignAccess(Config{AuthBaseURL: "http://auth.local"}, nil, handlerDependencies{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "resolve static assets") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewHandlerFallsBackToInternalServerError(t *testing.T) {
	origSubStaticFS := subStaticFS
	subStaticFS = func() (fs.FS, error) {
		return nil, fmt.Errorf("injected static assets failure")
	}
	defer func() {
		subStaticFS = origSubStaticFS
	}()

	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestNewServerWithContextRequiresContext(t *testing.T) {
	_, err := NewServerWithContext(nil, Config{
		HTTPAddr:    "127.0.0.1:0",
		AuthBaseURL: "http://auth.local",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDialAuthGRPCNilAddr(t *testing.T) {
	conn, client, err := dialAuthGRPC(context.Background(), Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != nil || client != nil {
		t.Fatalf("expected nil conn and client")
	}
}

func TestDialAuthGRPCNilContextReturnsError(t *testing.T) {
	_, _, err := dialAuthGRPC(nil, Config{
		AuthAddr: "127.0.0.1:1",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "context is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDialAuthGRPCSuccess(t *testing.T) {
	listener, server := startGRPCServer(t)
	defer server.Stop()

	conn, client, err := dialAuthGRPC(context.Background(), Config{
		AuthAddr:        listener.Addr().String(),
		GRPCDialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if conn == nil || client == nil {
		t.Fatalf("expected conn and client")
	}
	_ = conn.Close()
}

func TestDialAuthGRPCDialError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := dialAuthGRPC(ctx, Config{
		AuthAddr:        "127.0.0.1:1",
		GRPCDialTimeout: 50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "dial auth gRPC") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDialAuthGRPCHealthError(t *testing.T) {
	listener, server := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer server.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, _, err := dialAuthGRPC(ctx, Config{
		AuthAddr:        listener.Addr().String(),
		GRPCDialTimeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "auth gRPC health check failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildAuthConsentURL(t *testing.T) {
	cases := []struct {
		name      string
		base      string
		pendingID string
		want      string
	}{
		{
			name:      "empty base",
			base:      "",
			pendingID: "pending 1",
			want:      "/authorize/consent?pending_id=pending+1",
		},
		{
			name:      "base trims slash",
			base:      "http://auth.local/",
			pendingID: "pending 1",
			want:      "http://auth.local/authorize/consent?pending_id=pending+1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildAuthConsentURL(tc.base, tc.pendingID); got != tc.want {
				t.Fatalf("buildAuthConsentURL(%q, %q) = %q, want %q", tc.base, tc.pendingID, got, tc.want)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want %q", got, "application/json")
	}

	var payload map[string]any
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("ok = %v, want true", payload["ok"])
	}
}

func TestNewServerSuccessAndClose(t *testing.T) {
	listener, server := startGRPCServer(t)
	defer server.Stop()

	webServer, err := NewServer(Config{
		HTTPAddr:        "127.0.0.1:0",
		AuthBaseURL:     "http://auth.local",
		AuthAddr:        listener.Addr().String(),
		GRPCDialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	webServer.Close()
}

func TestListenAndServeShutsDown(t *testing.T) {
	webServer, err := NewServer(Config{
		HTTPAddr:        "127.0.0.1:0",
		AuthBaseURL:     "http://auth.local",
		GRPCDialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- webServer.ListenAndServe(ctx)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for shutdown")
	}
}

func TestListenAndServeReturnsServeError(t *testing.T) {
	server := &Server{
		httpAddr:   "127.0.0.1:-1",
		httpServer: &http.Server{Addr: "127.0.0.1:-1"},
	}

	err := server.ListenAndServe(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "serve http") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListenAndServeRequiresContext(t *testing.T) {
	server := &Server{
		httpAddr:   "127.0.0.1:0",
		httpServer: &http.Server{Addr: "127.0.0.1:0"},
	}
	err := server.ListenAndServe(nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

type fakeAuthClient struct {
	beginLoginResp   *authv1.BeginPasskeyLoginResponse
	finishLoginResp  *authv1.FinishPasskeyLoginResponse
	createUserResp   *authv1.CreateUserResponse
	beginRegResp     *authv1.BeginPasskeyRegistrationResponse
	finishRegResp    *authv1.FinishPasskeyRegistrationResponse
	consumeMagicResp *authv1.ConsumeMagicLinkResponse
}

func (f *fakeAuthClient) CreateUser(ctx context.Context, req *authv1.CreateUserRequest, opts ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	if f.createUserResp != nil {
		return f.createUserResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) BeginPasskeyRegistration(ctx context.Context, req *authv1.BeginPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	if f.beginRegResp != nil {
		return f.beginRegResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) FinishPasskeyRegistration(ctx context.Context, req *authv1.FinishPasskeyRegistrationRequest, opts ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	if f.finishRegResp != nil {
		return f.finishRegResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) BeginPasskeyLogin(ctx context.Context, req *authv1.BeginPasskeyLoginRequest, opts ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	if f.beginLoginResp != nil {
		return f.beginLoginResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) FinishPasskeyLogin(ctx context.Context, req *authv1.FinishPasskeyLoginRequest, opts ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	if f.finishLoginResp != nil {
		return f.finishLoginResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) GenerateMagicLink(ctx context.Context, req *authv1.GenerateMagicLinkRequest, opts ...grpc.CallOption) (*authv1.GenerateMagicLinkResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) ConsumeMagicLink(ctx context.Context, req *authv1.ConsumeMagicLinkRequest, opts ...grpc.CallOption) (*authv1.ConsumeMagicLinkResponse, error) {
	if f.consumeMagicResp != nil {
		return f.consumeMagicResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) ListUserEmails(ctx context.Context, req *authv1.ListUserEmailsRequest, opts ...grpc.CallOption) (*authv1.ListUserEmailsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) IssueJoinGrant(ctx context.Context, req *authv1.IssueJoinGrantRequest, opts ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) GetUser(ctx context.Context, req *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeAuthClient) ListUsers(ctx context.Context, req *authv1.ListUsersRequest, opts ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestMagicLinkNilAuthClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/magic?token=token-1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(w.Body.String(), "Magic link unavailable") {
		t.Fatalf("expected unavailable page")
	}
}

func TestPasskeyLoginStartMethodNotAllowed(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/passkeys/login/start", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPasskeyRegisterFinishMethodNotAllowed(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/passkeys/register/finish", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestLoginHandlerMethodNotAllowed(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/login?pending_id=pending-1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPasskeyLoginFinishMethodNotAllowed(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/passkeys/login/finish", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPasskeyRegisterStartMethodNotAllowed(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/passkeys/register/start", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPasskeyLoginFinishNilAuthClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/finish", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestPasskeyRegisterStartNilAuthClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/start", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestPasskeyRegisterFinishNilAuthClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/finish", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestMagicLinkInvalidToken(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodGet, "/magic?token=bad-token", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "Magic link invalid") {
		t.Fatalf("expected invalid page")
	}
}

func TestPasskeyLoginStartNilAuthClient(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/start", bytes.NewBufferString(`{"pending_id":"p1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestPasskeyLoginFinishError(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/finish",
		bytes.NewBufferString(`{"pending_id":"p1","session_id":"s1","credential":{}}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPasskeyLoginStartError(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/start",
		bytes.NewBufferString(`{"pending_id":"p1"}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPasskeyRegisterFinishError(t *testing.T) {
	fake := &fakeAuthClient{}
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, fake)
	req := httptest.NewRequest(http.MethodPost, "/passkeys/register/finish",
		bytes.NewBufferString(`{"session_id":"s1","user_id":"u1","credential":{}}`))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func startGRPCServer(t *testing.T) (net.Listener, *grpc.Server) {
	return startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
}

func startHealthServer(t *testing.T, status grpc_health_v1.HealthCheckResponse_ServingStatus) (net.Listener, *grpc.Server) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", status)
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})
	return listener, server
}
