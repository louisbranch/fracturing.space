package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeCampaignAccessChecker struct {
	allowed bool
	err     error
}

func (f fakeCampaignAccessChecker) IsCampaignParticipant(_ context.Context, _ string, _ string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.allowed, nil
}

func TestCampaignPageRedirectsUnauthenticatedToLogin(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123", nil)
	w := httptest.NewRecorder()

	h.handleCampaignPage(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestCampaignPageForbiddenForNonParticipant(t *testing.T) {
	h := &handler{
		config:         Config{AuthBaseURL: "http://auth.local"},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignAccess: fakeCampaignAccessChecker{allowed: false},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleCampaignPage(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCampaignPageParticipantRendersWithSharedLayout(t *testing.T) {
	h := &handler{
		config:         Config{AuthBaseURL: "http://auth.local"},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignAccess: fakeCampaignAccessChecker{allowed: true},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleCampaignPage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, `class="landing-body"`) {
		t.Fatalf("expected shared layout body class")
	}
	if !strings.Contains(body, "Campaign camp-123") {
		t.Fatalf("expected campaign heading")
	}
	if !strings.Contains(body, "chat.join") {
		t.Fatalf("expected phase 1 chat protocol usage in page script")
	}
}

func TestCampaignPageReturnsBadGatewayOnAccessCheckerError(t *testing.T) {
	h := &handler{
		config:         Config{AuthBaseURL: "http://auth.local"},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignAccess: fakeCampaignAccessChecker{err: errors.New("upstream failure")},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleCampaignPage(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadGateway)
	}
}

func TestCampaignPageRejectsNonGET(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestCampaignPageRejectsInvalidPath(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/extra", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
