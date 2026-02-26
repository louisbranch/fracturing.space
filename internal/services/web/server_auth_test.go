package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginCookieAllowsProtectedRoute(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(defaultProtectedConfig(newFakeWebAuthClient()))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	payload := map[string]any{"session_id": "session-1", "credential": map[string]any{"id": "cred-1"}}
	body, _ := json.Marshal(payload)
	loginReq := httptest.NewRequest(http.MethodPost, "/passkeys/login/finish", bytes.NewReader(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	h.ServeHTTP(loginRR, loginReq)
	if loginRR.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginRR.Code, http.StatusOK)
	}
	setCookie := loginRR.Header().Get("Set-Cookie")
	cookie, err := http.ParseSetCookie(setCookie)
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	if cookie.Name != "web_session" {
		t.Fatalf("cookie name = %q, want %q", cookie.Name, "web_session")
	}
	if strings.TrimSpace(cookie.Value) == "" {
		t.Fatalf("expected non-empty session cookie value")
	}

	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/", nil)
	req.Header.Set("Cookie", setCookie)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestLogoutCookieRelocksProtectedRoute(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(defaultProtectedConfig(newFakeWebAuthClient()))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	payload := map[string]any{"session_id": "session-1", "credential": map[string]any{"id": "cred-1"}}
	body, _ := json.Marshal(payload)
	loginReq := httptest.NewRequest(http.MethodPost, "/passkeys/login/finish", bytes.NewReader(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	h.ServeHTTP(loginRR, loginReq)
	if loginRR.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginRR.Code, http.StatusOK)
	}
	setCookie := loginRR.Header().Get("Set-Cookie")

	logoutReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logoutReq.Header.Set("Cookie", setCookie)
	logoutReq.Header.Set("Origin", "http://example.com")
	logoutRR := httptest.NewRecorder()
	h.ServeHTTP(logoutRR, logoutReq)
	if logoutRR.Code != http.StatusFound {
		t.Fatalf("logout status = %d, want %d", logoutRR.Code, http.StatusFound)
	}

	clearedCookie := logoutRR.Header().Get("Set-Cookie")
	cleared, err := http.ParseSetCookie(clearedCookie)
	if err != nil {
		t.Fatalf("ParseSetCookie() error = %v", err)
	}
	if cleared.Name != "web_session" {
		t.Fatalf("cookie name = %q, want %q", cleared.Name, "web_session")
	}
	if cleared.MaxAge > 0 {
		t.Fatalf("cookie max-age = %d, want immediate expiration", cleared.MaxAge)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/", nil)
	req.Header.Set("Cookie", clearedCookie)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want %q", got, "/login")
	}
}

func TestProtectedRouteRejectsUnknownSessionCookie(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(defaultProtectedConfig(newFakeWebAuthClient()))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/", nil)
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "missing-session"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want %q", got, "/login")
	}
}

func TestLogoutRevokesPreviouslyIssuedSessionCookie(t *testing.T) {
	t.Parallel()

	h, err := NewHandler(defaultProtectedConfig(newFakeWebAuthClient()))
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	payload := map[string]any{"session_id": "session-1", "credential": map[string]any{"id": "cred-1"}}
	body, _ := json.Marshal(payload)
	loginReq := httptest.NewRequest(http.MethodPost, "/passkeys/login/finish", bytes.NewReader(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	h.ServeHTTP(loginRR, loginReq)
	if loginRR.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginRR.Code, http.StatusOK)
	}
	originalCookie := loginRR.Header().Get("Set-Cookie")

	logoutReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logoutReq.Header.Set("Cookie", originalCookie)
	logoutReq.Header.Set("Origin", "http://example.com")
	logoutRR := httptest.NewRecorder()
	h.ServeHTTP(logoutRR, logoutReq)
	if logoutRR.Code != http.StatusFound {
		t.Fatalf("logout status = %d, want %d", logoutRR.Code, http.StatusFound)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/", nil)
	req.Header.Set("Cookie", originalCookie)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want %q", got, "/login")
	}
}
