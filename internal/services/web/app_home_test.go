package web

import (
	"net/http"
	"net/http/httptest"
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

func TestAppHomeHandlerRedirectsAuthenticatedToCampaigns(t *testing.T) {
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
