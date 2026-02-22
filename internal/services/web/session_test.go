package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type countingPersistenceStore struct {
	inner          sessionPersistence
	totalLoadCount int32
	mu             sync.Mutex
}

func (c *countingPersistenceStore) LoadSession(ctx context.Context, sessionID string) (string, string, time.Time, bool, error) {
	c.mu.Lock()
	c.totalLoadCount++
	c.mu.Unlock()
	return c.inner.LoadSession(ctx, sessionID)
}

func (c *countingPersistenceStore) SaveSession(ctx context.Context, sessionID, accessToken, displayName string, expiresAt time.Time) error {
	if c == nil {
		return nil
	}
	return c.inner.SaveSession(ctx, sessionID, accessToken, displayName, expiresAt)
}

func (c *countingPersistenceStore) DeleteSession(ctx context.Context, sessionID string) error {
	if c == nil {
		return nil
	}
	return c.inner.DeleteSession(ctx, sessionID)
}

func (c *countingPersistenceStore) loadCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return int(c.totalLoadCount)
}

func TestSessionStoreCreateAndGet(t *testing.T) {
	store := newSessionStore()
	id := store.create("token-1", "Alice", time.Now().Add(time.Hour))
	if id == "" {
		t.Fatal("expected non-empty session ID")
	}

	sess := store.get(id, "")
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
	if got := store.get(id, ""); got != nil {
		t.Fatal("expected nil for expired session")
	}
}

func TestSessionStoreGetMissing(t *testing.T) {
	store := newSessionStore()
	if got := store.get("nonexistent", ""); got != nil {
		t.Fatal("expected nil for missing session")
	}
}

func TestSessionStoreDelete(t *testing.T) {
	store := newSessionStore()
	id := store.create("token-1", "Alice", time.Now().Add(time.Hour))
	store.delete(id)
	if got := store.get(id, ""); got != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestSessionStoreRestoreFromPersistentStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web-cache.db")
	persistentStore, err := openWebCacheStore(path)
	if err != nil {
		t.Fatalf("open web cache store: %v", err)
	}
	if err := persistentStore.SaveSession(context.Background(), "persistent-session", "token-1", "Alice", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("save session: %v", err)
	}
	if err := persistentStore.Close(); err != nil {
		t.Fatalf("close web cache store: %v", err)
	}

	reopenedStore, err := openWebCacheStore(path)
	if err != nil {
		t.Fatalf("reopen web cache store: %v", err)
	}
	t.Cleanup(func() {
		if err := reopenedStore.Close(); err != nil {
			t.Fatalf("close reopened web cache store: %v", err)
		}
	})
	sessionStore := newSessionStore(reopenedStore)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "persistent-session"})
	req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: "token-1"})

	sess := sessionFromRequest(req, sessionStore)
	if sess == nil {
		t.Fatal("expected session from persistent store")
	}
	if sess.accessToken != "token-1" {
		t.Fatalf("accessToken = %q, want %q", sess.accessToken, "token-1")
	}
	if sess.displayName != "Alice" {
		t.Fatalf("displayName = %q, want %q", sess.displayName, "Alice")
	}
}

func TestSessionStoreRestoreFromPersistentStoreRejectsMismatchedToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web-cache.db")
	persistentStore, err := openWebCacheStore(path)
	if err != nil {
		t.Fatalf("open web cache store: %v", err)
	}
	if err := persistentStore.SaveSession(context.Background(), "persistent-session", "token-1", "Alice", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("save session: %v", err)
	}
	if err := persistentStore.Close(); err != nil {
		t.Fatalf("close web cache store: %v", err)
	}

	reopenedStore, err := openWebCacheStore(path)
	if err != nil {
		t.Fatalf("reopen web cache store: %v", err)
	}
	t.Cleanup(func() {
		if err := reopenedStore.Close(); err != nil {
			t.Fatalf("close reopened web cache store: %v", err)
		}
	})
	sessionStore := newSessionStore(reopenedStore)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "persistent-session"})
	req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: "wrong-token"})

	sess := sessionFromRequest(req, sessionStore)
	if sess != nil {
		t.Fatal("expected nil for mismatched persisted session token")
	}

	_, _, _, found, err := reopenedStore.LoadSession(context.Background(), "persistent-session")
	if err != nil {
		t.Fatalf("load persisted session: %v", err)
	}
	if found {
		t.Fatal("expected mismatched session to be deleted")
	}
}

func TestSessionStoreExpiredFromPersistentStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web-cache.db")
	persistentStore, err := openWebCacheStore(path)
	if err != nil {
		t.Fatalf("open web cache store: %v", err)
	}
	if err := persistentStore.SaveSession(context.Background(), "expired-session", "token-1", "Alice", time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("save session: %v", err)
	}
	if err := persistentStore.Close(); err != nil {
		t.Fatalf("close web cache store: %v", err)
	}

	reopenedStore, err := openWebCacheStore(path)
	if err != nil {
		t.Fatalf("reopen web cache store: %v", err)
	}
	t.Cleanup(func() {
		if err := reopenedStore.Close(); err != nil {
			t.Fatalf("close reopened web cache store: %v", err)
		}
	})
	sessionStore := newSessionStore(reopenedStore)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "expired-session"})
	req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: "token-1"})

	sess := sessionFromRequest(req, sessionStore)
	if sess != nil {
		t.Fatal("expected nil session for expired persistent entry")
	}
}

func TestSessionStoreDeleteRemovesPersistentSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web-cache.db")
	persistentStore, err := openWebCacheStore(path)
	if err != nil {
		t.Fatalf("open web cache store: %v", err)
	}
	t.Cleanup(func() {
		if err := persistentStore.Close(); err != nil {
			t.Fatalf("close web cache store: %v", err)
		}
	})
	sessionStore := newSessionStore(persistentStore)
	sessionID := sessionStore.create("token-1", "Alice", time.Now().Add(time.Hour))

	sessionStore.delete(sessionID)
	_, _, _, found, err := persistentStore.LoadSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("load deleted session: %v", err)
	}
	if found {
		t.Fatal("expected session to be deleted from persistent store")
	}
}

func TestSessionStoreConcurrentRestoreLoadsOnlyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web-cache.db")
	persistentStore, err := openWebCacheStore(path)
	if err != nil {
		t.Fatalf("open web cache store: %v", err)
	}
	t.Cleanup(func() {
		if err := persistentStore.Close(); err != nil {
			t.Fatalf("close web cache store: %v", err)
		}
	})
	if err := persistentStore.SaveSession(context.Background(), "persistent-session", "token-1", "Alice", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("save session: %v", err)
	}

	wrapped := &countingPersistenceStore{inner: persistentStore}
	sessionStore := newSessionStore(wrapped)

	const concurrentReaders = 24
	var missing int32
	var done sync.WaitGroup
	done.Add(concurrentReaders)
	for i := 0; i < concurrentReaders; i++ {
		go func() {
			defer done.Done()
			sess := sessionStore.get("persistent-session", "token-1")
			if sess == nil {
				atomic.StoreInt32(&missing, 1)
			}
		}()
	}
	done.Wait()
	if atomic.LoadInt32(&missing) != 0 {
		t.Fatal("expected persisted session to restore under concurrent reads")
	}
	if got := wrapped.loadCount(); got != 1 {
		t.Fatalf("LoadSession calls = %d, want 1", got)
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

func TestTokenCookieDomainForRequest(t *testing.T) {
	tests := []struct {
		name         string
		configDomain string
		requestHost  string
		want         string
	}{
		{
			name:         "empty config uses localhost for localhost host",
			configDomain: "",
			requestHost:  "localhost:8080",
			want:         "localhost",
		},
		{
			name:         "empty config uses localhost for localhost subdomain",
			configDomain: "",
			requestHost:  "chat.localhost",
			want:         "localhost",
		},
		{
			name:         "empty config uses localhost for loopback IPv4",
			configDomain: "",
			requestHost:  "127.0.0.1:8086",
			want:         "localhost",
		},
		{
			name:         "empty config uses localhost for loopback IPv6",
			configDomain: "",
			requestHost:  "[::1]:8086",
			want:         "localhost",
		},
		{
			name:         "empty config empty when host is not local",
			configDomain: "",
			requestHost:  "example.com",
			want:         "",
		},
		{
			name:         "configured domain for parent domain match",
			configDomain: "Example.Com",
			requestHost:  "chat.example.com:8080",
			want:         "example.com",
		},
		{
			name:         "configured domain for exact match",
			configDomain: "example.com",
			requestHost:  "example.com",
			want:         "example.com",
		},
		{
			name:         "configured domain mismatch",
			configDomain: "example.com",
			requestHost:  "localhost:8080",
			want:         "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			domain := tokenCookieDomainForRequest(tc.configDomain, tc.requestHost)
			if domain != tc.want {
				t.Fatalf("tokenCookieDomainForRequest(%q, %q) = %q, want %q", tc.configDomain, tc.requestHost, domain, tc.want)
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
