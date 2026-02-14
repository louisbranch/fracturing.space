package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionStoreCreateAndGet(t *testing.T) {
	store := newSessionStore()
	id := store.create("token-1", "Alice", time.Now().Add(time.Hour))
	if id == "" {
		t.Fatal("expected non-empty session ID")
	}

	sess := store.get(id)
	if sess == nil {
		t.Fatal("expected session")
	}
	if sess.accessToken != "token-1" {
		t.Fatalf("accessToken = %q, want %q", sess.accessToken, "token-1")
	}
	if sess.displayName != "Alice" {
		t.Fatalf("displayName = %q, want %q", sess.displayName, "Alice")
	}
}

func TestSessionStoreGetExpired(t *testing.T) {
	store := newSessionStore()
	id := store.create("token-1", "Alice", time.Now().Add(-time.Second))
	if got := store.get(id); got != nil {
		t.Fatal("expected nil for expired session")
	}
}

func TestSessionStoreGetMissing(t *testing.T) {
	store := newSessionStore()
	if got := store.get("nonexistent"); got != nil {
		t.Fatal("expected nil for missing session")
	}
}

func TestSessionStoreDelete(t *testing.T) {
	store := newSessionStore()
	id := store.create("token-1", "Alice", time.Now().Add(time.Hour))
	store.delete(id)
	if got := store.get(id); got != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestPendingFlowStoreCreateAndConsume(t *testing.T) {
	store := newPendingFlowStore()
	state := store.create("verifier-1")
	if state == "" {
		t.Fatal("expected non-empty state")
	}

	flow := store.consume(state)
	if flow == nil {
		t.Fatal("expected pending flow")
	}
	if flow.codeVerifier != "verifier-1" {
		t.Fatalf("codeVerifier = %q, want %q", flow.codeVerifier, "verifier-1")
	}

	// Second consume returns nil (already consumed).
	if got := store.consume(state); got != nil {
		t.Fatal("expected nil on second consume")
	}
}

func TestPendingFlowStoreConsumeExpired(t *testing.T) {
	store := newPendingFlowStore()
	store.ttl = time.Nanosecond
	state := store.create("verifier-1")
	time.Sleep(2 * time.Millisecond)
	if got := store.consume(state); got != nil {
		t.Fatal("expected nil for expired flow")
	}
}

func TestPendingFlowStoreConsumeMissing(t *testing.T) {
	store := newPendingFlowStore()
	if got := store.consume("nonexistent"); got != nil {
		t.Fatal("expected nil for missing flow")
	}
}

func TestSessionCookieHelpers(t *testing.T) {
	store := newSessionStore()
	id := store.create("token-1", "Alice", time.Now().Add(time.Hour))

	w := httptest.NewRecorder()
	setSessionCookie(w, id)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != sessionCookieName {
		t.Fatalf("cookie name = %q, want %q", cookies[0].Name, sessionCookieName)
	}
	if cookies[0].Value != id {
		t.Fatalf("cookie value = %q, want %q", cookies[0].Value, id)
	}

	// sessionFromRequest should retrieve the session.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookies[0])
	sess := sessionFromRequest(req, store)
	if sess == nil {
		t.Fatal("expected session from cookie")
	}
	if sess.displayName != "Alice" {
		t.Fatalf("displayName = %q, want %q", sess.displayName, "Alice")
	}
}

func TestSessionFromRequestNoCookie(t *testing.T) {
	store := newSessionStore()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := sessionFromRequest(req, store); got != nil {
		t.Fatal("expected nil without cookie")
	}
}

func TestClearSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	clearSessionCookie(w)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].MaxAge != -1 {
		t.Fatalf("MaxAge = %d, want -1", cookies[0].MaxAge)
	}
}

func TestSetTokenCookie(t *testing.T) {
	w := httptest.NewRecorder()
	setTokenCookie(w, "my-access-token", "example.com", 3600)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != tokenCookieName {
		t.Fatalf("cookie name = %q, want %q", c.Name, tokenCookieName)
	}
	if c.Value != "my-access-token" {
		t.Fatalf("cookie value = %q, want %q", c.Value, "my-access-token")
	}
	// Go's http.SetCookie strips the leading dot per RFC 6265; browsers still
	// scope to the domain and all subdomains when the Domain attribute is set.
	if c.Domain != "example.com" {
		t.Fatalf("Domain = %q, want %q", c.Domain, "example.com")
	}
	if !c.HttpOnly {
		t.Fatal("expected HttpOnly")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v, want Lax", c.SameSite)
	}
	if c.MaxAge != 3600 {
		t.Fatalf("MaxAge = %d, want 3600", c.MaxAge)
	}
	if c.Path != "/" {
		t.Fatalf("Path = %q, want %q", c.Path, "/")
	}
}

func TestSetTokenCookieDomainPrefix(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		// Go's http.SetCookie strips the leading dot per RFC 6265.
		{"example.com", "example.com"},
		{".example.com", "example.com"},
		{"localhost", "localhost"},
	}
	for _, tc := range tests {
		t.Run(tc.domain, func(t *testing.T) {
			w := httptest.NewRecorder()
			setTokenCookie(w, "tok", tc.domain, 60)
			c := w.Result().Cookies()[0]
			if c.Domain != tc.want {
				t.Fatalf("Domain = %q, want %q", c.Domain, tc.want)
			}
		})
	}
}

func TestClearTokenCookie(t *testing.T) {
	w := httptest.NewRecorder()
	clearTokenCookie(w, "example.com")
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	c := cookies[0]
	if c.Name != tokenCookieName {
		t.Fatalf("cookie name = %q, want %q", c.Name, tokenCookieName)
	}
	if c.MaxAge != -1 {
		t.Fatalf("MaxAge = %d, want -1", c.MaxAge)
	}
	if c.Domain != "example.com" {
		t.Fatalf("Domain = %q, want %q", c.Domain, "example.com")
	}
}
