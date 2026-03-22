package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
)

func TestRequireCookieSessionSameOriginPassesGETWithSession(t *testing.T) {
	t.Parallel()

	handler := requireCookieSessionSameOrigin(requestmeta.SchemePolicy{})(okHandler())
	r := httptest.NewRequest(http.MethodGet, "http://localhost/app/dashboard", nil)
	r.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: "session-1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("GET with session: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireCookieSessionSameOriginPassesPOSTWithSessionAndProof(t *testing.T) {
	t.Parallel()

	handler := requireCookieSessionSameOrigin(requestmeta.SchemePolicy{})(okHandler())
	r := httptest.NewRequest(http.MethodPost, "http://localhost/app/campaigns/create", nil)
	r.Host = "localhost"
	r.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: "session-1"})
	r.Header.Set("Origin", "http://localhost")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("POST with session+proof: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireCookieSessionSameOriginBlocksPOSTWithSessionNoProof(t *testing.T) {
	t.Parallel()

	handler := requireCookieSessionSameOrigin(requestmeta.SchemePolicy{})(okHandler())
	r := httptest.NewRequest(http.MethodPost, "http://localhost/app/campaigns/create", nil)
	r.Host = "localhost"
	r.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: "session-1"})
	// No Origin or Referer header — same-origin proof is missing.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("POST with session, no proof: status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireCookieSessionSameOriginPassesPOSTWithoutSession(t *testing.T) {
	t.Parallel()

	handler := requireCookieSessionSameOrigin(requestmeta.SchemePolicy{})(okHandler())
	r := httptest.NewRequest(http.MethodPost, "http://localhost/login", nil)
	// No session cookie — anonymous mutation is allowed.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("POST without session: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireCookieSessionSameOriginPassesDELETEWithSessionAndProof(t *testing.T) {
	t.Parallel()

	handler := requireCookieSessionSameOrigin(requestmeta.SchemePolicy{})(okHandler())
	r := httptest.NewRequest(http.MethodDelete, "http://localhost/app/settings/passkey", nil)
	r.Host = "localhost"
	r.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: "session-1"})
	r.Header.Set("Origin", "http://localhost")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE with session+proof: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireCookieSessionSameOriginBlocksCrossOriginPOST(t *testing.T) {
	t.Parallel()

	handler := requireCookieSessionSameOrigin(requestmeta.SchemePolicy{})(okHandler())
	r := httptest.NewRequest(http.MethodPost, "http://localhost/app/campaigns/create", nil)
	r.Host = "localhost"
	r.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: "session-1"})
	r.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin POST: status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireCookieSessionSameOriginPassesHEADWithSession(t *testing.T) {
	t.Parallel()

	handler := requireCookieSessionSameOrigin(requestmeta.SchemePolicy{})(okHandler())
	r := httptest.NewRequest(http.MethodHead, "http://localhost/app/dashboard", nil)
	r.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: "session-1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("HEAD with session: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
