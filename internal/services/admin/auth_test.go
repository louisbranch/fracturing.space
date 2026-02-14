package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPIntrospectorActiveToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer my-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer my-token")
		}
		if got := r.Header.Get("X-Resource-Secret"); got != "my-secret" {
			t.Errorf("X-Resource-Secret = %q, want %q", got, "my-secret")
		}
		// Auth service returns user_id, not sub.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"active":true,"user_id":"user-42","scope":"openid"}`))
	}))
	defer server.Close()

	intr := newHTTPIntrospector(server.URL, "my-secret")
	result, err := intr.Introspect(context.Background(), "my-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Active {
		t.Fatal("expected active = true")
	}
	if result.UserID != "user-42" {
		t.Fatalf("UserID = %q, want %q", result.UserID, "user-42")
	}
}

func TestHTTPIntrospectorInactiveToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: false})
	}))
	defer server.Close()

	intr := newHTTPIntrospector(server.URL, "my-secret")
	result, err := intr.Introspect(context.Background(), "my-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Active {
		t.Fatal("expected active = false")
	}
}

func TestHTTPIntrospectorNetworkError(t *testing.T) {
	// Use a server that's already closed.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	intr := newHTTPIntrospector(server.URL, "my-secret")
	_, err := intr.Introspect(context.Background(), "my-token")
	if err == nil {
		t.Fatal("expected error for closed server")
	}
}

func TestHTTPIntrospectorNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	intr := newHTTPIntrospector(server.URL, "my-secret")
	_, err := intr.Introspect(context.Background(), "my-token")
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

// fakeIntrospector is a test double for TokenIntrospector.
type fakeIntrospector struct {
	result introspectResponse
	err    error
}

func (f *fakeIntrospector) Introspect(_ context.Context, _ string) (introspectResponse, error) {
	return f.result, f.err
}

const testLoginURL = "http://login.example.com/auth/login"

func TestRequireAuthNoCookie(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})
	mw := requireAuth(inner, &fakeIntrospector{}, testLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != testLoginURL {
		t.Fatalf("Location = %q, want %q", loc, testLoginURL)
	}
}

func TestRequireAuthEmptyCookie(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})
	mw := requireAuth(inner, &fakeIntrospector{}, testLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "fs_token", Value: ""})
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestRequireAuthInactiveToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})
	intr := &fakeIntrospector{result: introspectResponse{Active: false}}
	mw := requireAuth(inner, intr, testLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "fs_token", Value: "expired-token"})
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestRequireAuthIntrospectError(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})
	intr := &fakeIntrospector{err: fmt.Errorf("connection refused")}
	mw := requireAuth(inner, intr, testLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "fs_token", Value: "some-token"})
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
}

func TestRequireAuthValidToken(t *testing.T) {
	var calledWithUserID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledWithUserID = authUserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	intr := &fakeIntrospector{result: introspectResponse{Active: true, UserID: "user-42"}}
	mw := requireAuth(inner, intr, testLoginURL)

	req := httptest.NewRequest(http.MethodGet, "/campaigns", nil)
	req.AddCookie(&http.Cookie{Name: "fs_token", Value: "valid-token"})
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if calledWithUserID != "user-42" {
		t.Fatalf("userID in context = %q, want %q", calledWithUserID, "user-42")
	}
}

func TestRequireAuthExemptPaths(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// Use nil introspector to prove it's never called.
	mw := requireAuth(inner, nil, testLoginURL)

	paths := []string{"/static/css/style.css", "/static/js/app.js"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			mw.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}
		})
	}
}
