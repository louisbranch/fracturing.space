package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAppHomeRouteRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppHomeRouteRedirectTargetIsRegistered(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)

	appReq := httptest.NewRequest(http.MethodGet, "/app", nil)
	appResp := httptest.NewRecorder()
	handler.ServeHTTP(appResp, appReq)

	if appResp.Code != http.StatusFound {
		t.Fatalf("/app status = %d, want %d", appResp.Code, http.StatusFound)
	}
	if location := appResp.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("/app location = %q, want %q", location, "/auth/login")
	}

	loginReq := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	loginResp := httptest.NewRecorder()
	handler.ServeHTTP(loginResp, loginReq)

	if loginResp.Code != http.StatusInternalServerError {
		t.Fatalf("/auth/login status = %d, want %d", loginResp.Code, http.StatusInternalServerError)
	}
}

func TestAppHomeRouteRejectsNonGET(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/app", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
	if allow := w.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", allow, http.MethodGet)
	}
}

func TestAppHomeHandlerRedirectsAuthenticatedToHomeShell(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppHome(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/app/campaigns" {
		t.Fatalf("location = %q, want %q", location, "/app/campaigns")
	}
}

func TestAppDashboardRouteRedirectsToHome(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/app", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppDashboard(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/app/campaigns" {
		t.Fatalf("location = %q, want %q", location, "/app/campaigns")
	}
}

func TestAppRootRendersAuthenticatedDashboard(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local", AppName: "Test App"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppRoot(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Test App") {
		t.Fatalf("body should include app name for dashboard branding")
	}
	if !strings.Contains(body, "/auth/logout") {
		t.Fatalf("body should include sign out form")
	}
}
