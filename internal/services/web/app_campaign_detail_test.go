package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAppCampaignDetailPageRedirectsUnauthenticatedToLogin(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123", nil)
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignDetailPageForbiddenForNonParticipant(t *testing.T) {
	h := &handler{
		config:         Config{AuthBaseURL: "http://auth.local"},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignAccess: fakeCampaignAccessChecker{allowed: false},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestAppCampaignDetailPageParticipantRendersCampaign(t *testing.T) {
	h := &handler{
		config:         Config{AuthBaseURL: "http://auth.local"},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignAccess: fakeCampaignAccessChecker{allowed: true},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Campaign camp-123") {
		t.Fatalf("expected campaign heading in body")
	}
	if !strings.Contains(body, "/app/campaigns/camp-123/sessions") {
		t.Fatalf("expected sessions link in body")
	}
	if !strings.Contains(body, "/app/campaigns/camp-123/participants") {
		t.Fatalf("expected participants link in body")
	}
	if !strings.Contains(body, "/app/campaigns/camp-123/characters") {
		t.Fatalf("expected characters link in body")
	}
	if !strings.Contains(body, "/app/campaigns/camp-123/invites") {
		t.Fatalf("expected invites link in body")
	}
}

func TestAppCampaignDetailPageReturnsBadGatewayOnAccessCheckerError(t *testing.T) {
	h := &handler{
		config:         Config{AuthBaseURL: "http://auth.local"},
		sessions:       newSessionStore(),
		pendingFlows:   newPendingFlowStore(),
		campaignAccess: fakeCampaignAccessChecker{err: errors.New("upstream failure")},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadGateway)
	}
}

func TestAppCampaignDetailPageRejectsInvalidPath(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/extra", nil)
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAppCampaignDetailPageRejectsNonGET(t *testing.T) {
	h := &handler{
		config:       Config{AuthBaseURL: "http://auth.local"},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123", nil)
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
	if allow := w.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", allow, http.MethodGet)
	}
}
