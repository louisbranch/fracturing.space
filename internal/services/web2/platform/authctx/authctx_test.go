package authctx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeaderOrCookieAuthRequiresIdentity(t *testing.T) {
	t.Parallel()

	auth := HeaderOrCookieAuth()
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/1", nil)
	if auth(req) {
		t.Fatalf("expected unauthenticated request")
	}
}

func TestHeaderOrCookieAuthRejectsHeaderIdentity(t *testing.T) {
	t.Parallel()

	auth := HeaderOrCookieAuth()
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/1", nil)
	req.Header.Set("X-Web2-User", "user-1")
	if auth(req) {
		t.Fatalf("expected header-only request to be rejected")
	}
}

func TestHeaderOrCookieAuthAcceptsSessionCookie(t *testing.T) {
	t.Parallel()

	auth := HeaderOrCookieAuth()
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/1", nil)
	req.AddCookie(&http.Cookie{Name: "web2_session", Value: "session-1"})
	if !auth(req) {
		t.Fatalf("expected authenticated request from cookie")
	}
}

func TestHeaderOrValidatedSessionAuthRejectsUnknownCookie(t *testing.T) {
	t.Parallel()

	auth := HeaderOrValidatedSessionAuth(func(context.Context, string) bool { return false })
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/1", nil)
	req.AddCookie(&http.Cookie{Name: "web2_session", Value: "missing"})
	if auth(req) {
		t.Fatalf("expected rejected unknown session")
	}
}

func TestHeaderOrValidatedSessionAuthAcceptsValidatedCookie(t *testing.T) {
	t.Parallel()

	auth := HeaderOrValidatedSessionAuth(func(_ context.Context, sid string) bool { return sid == "ws-1" })
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/1", nil)
	req.AddCookie(&http.Cookie{Name: "web2_session", Value: "ws-1"})
	if !auth(req) {
		t.Fatalf("expected validated session cookie")
	}
}

func TestHeaderOrValidatedSessionAuthRejectsHeaderIdentity(t *testing.T) {
	t.Parallel()

	auth := HeaderOrValidatedSessionAuth(func(context.Context, string) bool { return true })
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/1", nil)
	req.Header.Set("X-Web2-User", "user-1")
	if auth(req) {
		t.Fatalf("expected header-only request to be rejected")
	}
}

func TestValidatedSessionAuthRejectsHeaderOnlyIdentity(t *testing.T) {
	t.Parallel()

	auth := ValidatedSessionAuth(func(context.Context, string) bool { return true })
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/1", nil)
	req.Header.Set("X-Web2-User", "user-1")
	if auth(req) {
		t.Fatalf("expected header-only identity to be rejected")
	}
}

func TestValidatedSessionAuthAcceptsValidatedCookie(t *testing.T) {
	t.Parallel()

	auth := ValidatedSessionAuth(func(_ context.Context, sid string) bool { return sid == "ws-1" })
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/1", nil)
	req.AddCookie(&http.Cookie{Name: "web2_session", Value: "ws-1"})
	if !auth(req) {
		t.Fatalf("expected validated session cookie")
	}
}
