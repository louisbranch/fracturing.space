package oauth

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestHandleMetadataMethodNotAllowed(t *testing.T) {
	server := &Server{config: Config{Issuer: "https://example.com"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/.well-known/oauth-authorization-server", nil)

	server.handleMetadata(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleMetadataUsesConfigIssuer(t *testing.T) {
	server := &Server{config: Config{Issuer: "https://example.com/"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)

	server.handleMetadata(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var metadata AuthorizationServerMetadata
	if err := json.NewDecoder(rec.Body).Decode(&metadata); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if metadata.Issuer != "https://example.com" {
		t.Fatalf("Issuer = %q, want %q", metadata.Issuer, "https://example.com")
	}
	if metadata.AuthorizationEndpoint != "https://example.com/authorize" {
		t.Fatalf("AuthorizationEndpoint = %q", metadata.AuthorizationEndpoint)
	}
}

func TestHandleMetadataUsesRequestIssuer(t *testing.T) {
	server := &Server{config: Config{Issuer: ""}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.test/.well-known/oauth-authorization-server", nil)
	req.Host = "example.test"

	server.handleMetadata(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var metadata AuthorizationServerMetadata
	if err := json.NewDecoder(rec.Body).Decode(&metadata); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if metadata.Issuer != "http://example.test" {
		t.Fatalf("Issuer = %q, want %q", metadata.Issuer, "http://example.test")
	}
}

func TestTokenAuthMethodsSupported(t *testing.T) {
	methods := tokenAuthMethodsSupported([]Client{
		{ID: "one"},
		{ID: "two", Secret: "secret"},
	})
	if !reflect.DeepEqual(methods, []string{"none", "client_secret_post"}) {
		t.Fatalf("methods = %v, want %v", methods, []string{"none", "client_secret_post"})
	}
}

func TestIssuerFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.test", nil)
	req.Host = "example.test"
	if got := issuerFromRequest(req); got != "http://example.test" {
		t.Fatalf("issuerFromRequest() = %q, want %q", got, "http://example.test")
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.test", nil)
	req.Host = "example.test"
	req.TLS = &tls.ConnectionState{}
	if got := issuerFromRequest(req); got != "https://example.test" {
		t.Fatalf("issuerFromRequest(TLS) = %q, want %q", got, "https://example.test")
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.test", nil)
	req.Host = "example.test"
	req.Header.Set("X-Forwarded-Proto", "https")
	if got := issuerFromRequest(req); got != "https://example.test" {
		t.Fatalf("issuerFromRequest(forwarded) = %q, want %q", got, "https://example.test")
	}
}
