package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionResolverNilClientReturnsEmpty(t *testing.T) {
	t.Parallel()

	r := newSessionResolver(nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := r.resolveRequestUserID(req); got != "" {
		t.Fatalf("resolveRequestUserID = %q, want empty", got)
	}
	if r.resolveRequestSignedIn(req) {
		t.Fatalf("resolveRequestSignedIn = true, want false")
	}
}

func TestSessionResolverResolvesValidSession(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := newSessionResolver(auth)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-42")

	if got := r.resolveRequestUserID(req); got != "user-42" {
		t.Fatalf("resolveRequestUserID = %q, want %q", got, "user-42")
	}
	if !r.resolveRequestSignedIn(req) {
		t.Fatalf("resolveRequestSignedIn = false, want true")
	}
}

func TestSessionResolverMissingCookieNotSignedIn(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := newSessionResolver(auth)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if r.resolveRequestSignedIn(req) {
		t.Fatalf("resolveRequestSignedIn = true, want false without cookie")
	}
}

func TestSessionResolverUnknownSessionNotSignedIn(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := newSessionResolver(auth)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "unknown-session"})

	if r.resolveRequestSignedIn(req) {
		t.Fatalf("resolveRequestSignedIn = true, want false with unknown session")
	}
}

func TestSessionResolverAuthRequiredRejectsNoSession(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := newSessionResolver(auth)
	check := r.authRequired()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if check(req) {
		t.Fatalf("authRequired = true, want false without session")
	}
}

func TestSessionResolverAuthRequiredAcceptsValidSession(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := newSessionResolver(auth)
	check := r.authRequired()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")

	if !check(req) {
		t.Fatalf("authRequired = false, want true with valid session")
	}
}
