package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provideroauth"
)

func TestOAuthAdapterBuildAuthorizationURL(t *testing.T) {
	adapter := NewOAuthAdapter(OAuthConfig{
		AuthorizationURL: "https://provider.example.com/oauth/authorize",
		ClientID:         "client-1",
		RedirectURI:      "https://app.example.com/oauth/callback",
	})

	raw, err := adapter.BuildAuthorizationURL(provideroauth.AuthorizationURLInput{
		State:           "state-1",
		CodeChallenge:   "challenge-1",
		RequestedScopes: []string{"responses.read", "responses.write"},
	})
	if err != nil {
		t.Fatalf("build authorization url: %v", err)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	q := parsed.Query()
	if q.Get("response_type") != "code" {
		t.Fatalf("response_type = %q, want %q", q.Get("response_type"), "code")
	}
	if q.Get("client_id") != "client-1" {
		t.Fatalf("client_id = %q, want %q", q.Get("client_id"), "client-1")
	}
	if q.Get("redirect_uri") != "https://app.example.com/oauth/callback" {
		t.Fatalf("redirect_uri = %q, want callback uri", q.Get("redirect_uri"))
	}
	if q.Get("state") != "state-1" {
		t.Fatalf("state = %q, want %q", q.Get("state"), "state-1")
	}
	if q.Get("code_challenge") != "challenge-1" {
		t.Fatalf("code_challenge = %q, want %q", q.Get("code_challenge"), "challenge-1")
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("code_challenge_method = %q, want %q", q.Get("code_challenge_method"), "S256")
	}
	if q.Get("scope") != "responses.read responses.write" {
		t.Fatalf("scope = %q, want %q", q.Get("scope"), "responses.read responses.write")
	}
}

func TestOAuthAdapterExchangeAuthorizationCode(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Fatalf("grant_type = %q, want %q", r.Form.Get("grant_type"), "authorization_code")
		}
		if r.Form.Get("code") != "code-1" {
			t.Fatalf("code = %q, want %q", r.Form.Get("code"), "code-1")
		}
		if r.Form.Get("code_verifier") != "verifier-1" {
			t.Fatalf("code_verifier = %q, want %q", r.Form.Get("code_verifier"), "verifier-1")
		}
		if r.Form.Get("client_id") != "client-1" {
			t.Fatalf("client_id = %q, want %q", r.Form.Get("client_id"), "client-1")
		}
		if r.Form.Get("client_secret") != "secret-1" {
			t.Fatalf("client_secret = %q, want %q", r.Form.Get("client_secret"), "secret-1")
		}
		if r.Form.Get("redirect_uri") != "https://app.example.com/oauth/callback" {
			t.Fatalf("redirect_uri = %q, want callback uri", r.Form.Get("redirect_uri"))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "at-1",
			"refresh_token": "rt-1",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "responses.read responses.write",
		})
	}))
	defer tokenServer.Close()

	adapter := NewOAuthAdapter(OAuthConfig{
		TokenURL:     tokenServer.URL,
		ClientID:     "client-1",
		ClientSecret: "secret-1",
		RedirectURI:  "https://app.example.com/oauth/callback",
	})

	got, err := adapter.ExchangeAuthorizationCode(context.Background(), provideroauth.AuthorizationCodeInput{
		AuthorizationCode: "code-1",
		CodeVerifier:      "verifier-1",
	})
	if err != nil {
		t.Fatalf("exchange authorization code: %v", err)
	}
	if got.TokenPayload.AccessToken != "at-1" {
		t.Fatalf("access token = %q, want %q", got.TokenPayload.AccessToken, "at-1")
	}
	if strings.Contains(got.TokenPayload.AccessToken, "code-1") {
		t.Fatalf("access token must not contain authorization code: %q", got.TokenPayload.AccessToken)
	}
	if got.TokenPayload.RefreshToken != "rt-1" {
		t.Fatalf("refresh token = %q, want %q", got.TokenPayload.RefreshToken, "rt-1")
	}
	if got.ExpiresAt == nil {
		t.Fatal("expires_at is nil")
	}
}

func TestOAuthAdapterRefreshToken(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Fatalf("grant_type = %q, want %q", r.Form.Get("grant_type"), "refresh_token")
		}
		if r.Form.Get("refresh_token") != "rt-1" {
			t.Fatalf("refresh_token = %q, want %q", r.Form.Get("refresh_token"), "rt-1")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "at-2",
			"refresh_token": "rt-2",
			"token_type":    "Bearer",
			"expires_in":    1800,
			"scope":         "responses.read",
		})
	}))
	defer tokenServer.Close()

	adapter := NewOAuthAdapter(OAuthConfig{
		TokenURL:     tokenServer.URL,
		ClientID:     "client-1",
		ClientSecret: "secret-1",
	})
	got, err := adapter.RefreshToken(context.Background(), provideroauth.RefreshTokenInput{
		RefreshToken: "rt-1",
	})
	if err != nil {
		t.Fatalf("refresh token: %v", err)
	}
	if got.TokenPayload.AccessToken != "at-2" {
		t.Fatalf("access token = %q, want %q", got.TokenPayload.AccessToken, "at-2")
	}
	if got.ExpiresAt == nil || got.ExpiresAt.Before(time.Now().UTC()) {
		t.Fatalf("expires_at = %v, want future timestamp", got.ExpiresAt)
	}
}

func TestOAuthAdapterExchangeAuthorizationCodeNon2xxReadError(t *testing.T) {
	adapter := &oauthAdapter{cfg: OAuthConfig{
		TokenURL:     "https://provider.example.com/token",
		ClientID:     "client-1",
		ClientSecret: "secret-1",
		RedirectURI:  "https://app.example.com/oauth/callback",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header:     make(http.Header),
					Body:       failingReadCloser{},
				}, nil
			}),
		},
	}}

	_, err := adapter.ExchangeAuthorizationCode(context.Background(), provideroauth.AuthorizationCodeInput{
		AuthorizationCode: "code-1",
		CodeVerifier:      "verifier-1",
	})
	if err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("error = %v, want read error", err)
	}
}
