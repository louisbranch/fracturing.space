package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

func TestSessionResolverNilClientReturnsEmpty(t *testing.T) {
	t.Parallel()

	r := principal.New(principal.Dependencies{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := r.ResolveUserID(req); got != "" {
		t.Fatalf("ResolveUserID = %q, want empty", got)
	}
	if r.ResolveSignedIn(req) {
		t.Fatalf("ResolveSignedIn = true, want false")
	}
}

func TestSessionResolverResolvesValidSession(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{SessionClient: auth})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-42")

	if got := r.ResolveUserID(req); got != "user-42" {
		t.Fatalf("ResolveUserID = %q, want %q", got, "user-42")
	}
	if !r.ResolveSignedIn(req) {
		t.Fatalf("ResolveSignedIn = false, want true")
	}
}

func TestSessionResolverMissingCookieNotSignedIn(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{SessionClient: auth})
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if r.ResolveSignedIn(req) {
		t.Fatalf("ResolveSignedIn = true, want false without cookie")
	}
}

func TestSessionResolverUnknownSessionNotSignedIn(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{SessionClient: auth})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "web_session", Value: "unknown-session"})

	if r.ResolveSignedIn(req) {
		t.Fatalf("ResolveSignedIn = true, want false with unknown session")
	}
}

func TestSessionResolverAuthRequiredRejectsNoSession(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{SessionClient: auth})
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if r.AuthRequired(req) {
		t.Fatalf("AuthRequired = true, want false without session")
	}
}

func TestSessionResolverAuthRequiredAcceptsValidSession(t *testing.T) {
	t.Parallel()

	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{SessionClient: auth})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")

	if !r.AuthRequired(req) {
		t.Fatalf("AuthRequired = false, want true with valid session")
	}
}
