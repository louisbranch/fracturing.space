package authctx

import (
	"context"
	"encoding/json"
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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(IntrospectionResult{Active: true, UserID: "user-42"})
	}))
	defer server.Close()

	intr := NewHTTPIntrospector(server.URL, "my-secret", nil)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(IntrospectionResult{Active: false})
	}))
	defer server.Close()

	intr := NewHTTPIntrospector(server.URL, "my-secret", nil)
	result, err := intr.Introspect(context.Background(), "my-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Active {
		t.Fatal("expected active = false")
	}
}

func TestHTTPIntrospectorNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()

	intr := NewHTTPIntrospector(server.URL, "my-secret", nil)
	if _, err := intr.Introspect(context.Background(), "my-token"); err == nil {
		t.Fatal("expected error for closed server")
	}
}

func TestHTTPIntrospectorNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	intr := NewHTTPIntrospector(server.URL, "my-secret", nil)
	if _, err := intr.Introspect(context.Background(), "my-token"); err == nil {
		t.Fatal("expected error for non-200 status")
	}
}
