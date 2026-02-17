package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOpenAIOAuthAdapterBuildAuthorizationURL(t *testing.T) {
	adapter := NewOpenAIOAuthAdapter(OpenAIOAuthConfig{
		AuthorizationURL: "https://provider.example.com/oauth/authorize",
		ClientID:         "client-1",
		RedirectURI:      "https://app.example.com/oauth/callback",
	})

	raw, err := adapter.BuildAuthorizationURL(ProviderAuthorizationURLInput{
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

func TestOpenAIOAuthAdapterExchangeAuthorizationCode(t *testing.T) {
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

	adapter := NewOpenAIOAuthAdapter(OpenAIOAuthConfig{
		TokenURL:     tokenServer.URL,
		ClientID:     "client-1",
		ClientSecret: "secret-1",
		RedirectURI:  "https://app.example.com/oauth/callback",
	})

	got, err := adapter.ExchangeAuthorizationCode(context.Background(), ProviderAuthorizationCodeInput{
		AuthorizationCode: "code-1",
		CodeVerifier:      "verifier-1",
	})
	if err != nil {
		t.Fatalf("exchange authorization code: %v", err)
	}
	if got.TokenPlaintext == "" {
		t.Fatal("token plaintext is empty")
	}
	if strings.Contains(got.TokenPlaintext, "code-1") {
		t.Fatalf("token plaintext must not contain authorization code: %q", got.TokenPlaintext)
	}
	if !got.RefreshSupported {
		t.Fatal("refresh_supported = false, want true")
	}
	if got.ExpiresAt == nil {
		t.Fatal("expires_at is nil")
	}
}

func TestOpenAIOAuthAdapterRefreshToken(t *testing.T) {
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

	adapter := NewOpenAIOAuthAdapter(OpenAIOAuthConfig{
		TokenURL:     tokenServer.URL,
		ClientID:     "client-1",
		ClientSecret: "secret-1",
	})
	got, err := adapter.RefreshToken(context.Background(), ProviderRefreshTokenInput{
		RefreshToken: "rt-1",
	})
	if err != nil {
		t.Fatalf("refresh token: %v", err)
	}
	if got.TokenPlaintext == "" {
		t.Fatal("token plaintext is empty")
	}
	if got.ExpiresAt == nil || got.ExpiresAt.Before(time.Now().UTC()) {
		t.Fatalf("expires_at = %v, want future timestamp", got.ExpiresAt)
	}
}

func TestOpenAIOAuthAdapterExchangeAuthorizationCodeNon2xxReadError(t *testing.T) {
	adapter := &openAIOAuthAdapter{cfg: OpenAIOAuthConfig{
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

	_, err := adapter.ExchangeAuthorizationCode(context.Background(), ProviderAuthorizationCodeInput{
		AuthorizationCode: "code-1",
		CodeVerifier:      "verifier-1",
	})
	if err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("error = %v, want read error", err)
	}
}

func TestOpenAIInvokeAdapterInvoke(t *testing.T) {
	responsesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.Header.Get("Authorization") != "Bearer sk-1" {
			t.Fatalf("authorization = %q, want bearer token", r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["model"] != "gpt-4o-mini" {
			t.Fatalf("model = %v, want %q", body["model"], "gpt-4o-mini")
		}
		if body["input"] != "Say hello" {
			t.Fatalf("input = %v, want %q", body["input"], "Say hello")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output_text": "Hello from OpenAI",
		})
	}))
	defer responsesServer.Close()

	adapter := NewOpenAIInvokeAdapter(OpenAIInvokeConfig{
		ResponsesURL: responsesServer.URL,
	})
	got, err := adapter.Invoke(context.Background(), ProviderInvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		CredentialSecret: "sk-1",
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if got.OutputText != "Hello from OpenAI" {
		t.Fatalf("output_text = %q, want %q", got.OutputText, "Hello from OpenAI")
	}
}

func TestOpenAIInvokeAdapterInvokeFallbackOutput(t *testing.T) {
	responsesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{
				{
					"content": []map[string]any{
						{
							"type": "output_text",
							"text": "Fallback text",
						},
					},
				},
			},
		})
	}))
	defer responsesServer.Close()

	adapter := NewOpenAIInvokeAdapter(OpenAIInvokeConfig{
		ResponsesURL: responsesServer.URL,
	})
	got, err := adapter.Invoke(context.Background(), ProviderInvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		CredentialSecret: "sk-1",
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if got.OutputText != "Fallback text" {
		t.Fatalf("output_text = %q, want %q", got.OutputText, "Fallback text")
	}
}

func TestOpenAIInvokeAdapterInvokeReturnsErrorForNon2xx(t *testing.T) {
	responsesServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad credential", http.StatusUnauthorized)
	}))
	defer responsesServer.Close()

	adapter := NewOpenAIInvokeAdapter(OpenAIInvokeConfig{
		ResponsesURL: responsesServer.URL,
	})
	_, err := adapter.Invoke(context.Background(), ProviderInvokeInput{
		Model:            "gpt-4o-mini",
		Input:            "Say hello",
		CredentialSecret: "sk-1",
	})
	if err == nil {
		t.Fatal("expected invoke error")
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("error = %v, want status 401", err)
	}
}
