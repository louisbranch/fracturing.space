package service

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeRequestAuthorizer struct {
	calls int
	err   error
}

func (f *fakeRequestAuthorizer) Authorize(*http.Request) error {
	f.calls++
	return f.err
}

type fakeRateLimiter struct {
	calls int
	err   error
}

func (f *fakeRateLimiter) Allow(*http.Request) error {
	f.calls++
	return f.err
}

func TestApplyConfigUsesCustomRequestAuthorizer(t *testing.T) {
	transport := NewHTTPTransport("localhost:8081")
	customAuthorizer := &fakeRequestAuthorizer{}
	rateLimiter := &fakeRateLimiter{}
	cfg := Config{
		AuthToken:         "ignored-token",
		RequestAuthorizer: customAuthorizer,
		RateLimiter:       rateLimiter,
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
	}
	transport.applyConfig(cfg)

	if transport.requestAuthz != customAuthorizer {
		t.Fatalf("expected custom request authorizer to be used")
	}
	if transport.rateLimiter != rateLimiter {
		t.Fatalf("expected custom rate limiter to be used")
	}
	if transport.tlsConfig == nil || transport.tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("expected TLS config to be stored on transport")
	}
}

func TestApplyConfigBuildsHybridAuthorizerWhenTokenConfigured(t *testing.T) {
	transport := NewHTTPTransport("localhost:8081")
	transport.applyConfig(Config{AuthToken: "api-token"})

	authz, ok := transport.requestAuthz.(*hybridRequestAuthorizer)
	if !ok {
		t.Fatalf("expected hybridRequestAuthorizer, got %T", transport.requestAuthz)
	}
	if authz.apiToken != "api-token" {
		t.Fatalf("expected api token to be configured on hybrid authorizer")
	}
}

func TestApplyConfigOmitsTLSConfigWhenNotProvided(t *testing.T) {
	transport := NewHTTPTransport("localhost:8081")
	transport.tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	transport.applyConfig(Config{})

	if transport.tlsConfig != nil {
		t.Fatalf("expected TLS config to be cleared when not configured")
	}
}

func TestAuthorizeRequestRespectsRateLimiter(t *testing.T) {
	transport := NewHTTPTransport("localhost:8081")
	limiter := &fakeRateLimiter{err: errors.New("rate exceeded")}
	authorizer := &fakeRequestAuthorizer{}
	transport.applyConfig(Config{
		RequestAuthorizer: authorizer,
		RateLimiter:       limiter,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	allowed := transport.authorizeRequest(w, req)
	if allowed {
		t.Fatal("expected authorizeRequest to reject when rate limiter returns error")
	}
	if limiter.calls != 1 {
		t.Fatalf("expected 1 rate limiter call, got %d", limiter.calls)
	}
	if authorizer.calls != 0 {
		t.Fatalf("expected authorizer to be skipped when rate limiter rejects request")
	}
	if got := w.Result().StatusCode; got != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, got)
	}
}

func TestHybridRequestAuthorizerAllowsMatchingAPITokenWithoutOAuth(t *testing.T) {
	authorizationHits := 0
	oauth := &oauthAuth{
		issuer:         "https://example.test",
		resourceSecret: "secret",
		httpClient:     &http.Client{},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		authorizationHits++
	}))
	defer srv.Close()

	oauth.issuer = srv.URL
	authz := &hybridRequestAuthorizer{
		apiToken: "expected-token",
		oauth:    oauth,
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expected-token")

	if err := authz.Authorize(req); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if authorizationHits != 0 {
		t.Fatalf("expected OAuth introspection to be skipped when API token matches")
	}
}

func TestHybridRequestAuthorizerFallsBackToOAuthWhenAPIMiss(t *testing.T) {
	authorizationHits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		authorizationHits++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"active":true}`))
	}))
	defer srv.Close()

	authz := &hybridRequestAuthorizer{
		apiToken: "expected-token",
		oauth: &oauthAuth{
			issuer:         srv.URL,
			resourceSecret: "secret",
			httpClient:     srv.Client(),
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")

	if err := authz.Authorize(req); err != nil {
		t.Fatalf("expected fallback authorization to succeed, got %v", err)
	}
	if authorizationHits == 0 {
		t.Fatal("expected OAuth introspection to run on API token miss")
	}
}
