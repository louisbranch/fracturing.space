package server

import (
	"encoding/base64"
	"net"
	"path/filepath"
	"testing"
)

func setAISessionGrantEnv(t *testing.T) {
	t.Helper()
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER", "fracturing-space-game")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE", "fracturing-space-ai")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_TTL", "10m")
}

func TestNewRequiresEncryptionKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", "")
	setAISessionGrantEnv(t)

	if _, err := New(0); err == nil {
		t.Fatal("expected error for missing encryption key")
	}
}

func TestNewRejectsInvalidEncryptionKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", "not-base64")
	setAISessionGrantEnv(t)

	if _, err := New(0); err == nil {
		t.Fatal("expected error for invalid encryption key")
	}
}

func TestNewSuccess(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	srv, err := New(0)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	t.Cleanup(func() {
		srv.Close()
	})
	if srv.Addr() == "" {
		t.Fatal("expected non-empty address")
	}
}

func TestNewWithAddrContextRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	if _, err := NewWithAddrContext(nil, "127.0.0.1:0"); err == nil || err.Error() != "context is required" {
		t.Fatalf("NewWithAddrContext error = %v, want context is required", err)
	}
}

func TestRunWithAddrRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	if err := RunWithAddr(nil, "127.0.0.1:0"); err == nil || err.Error() != "context is required" {
		t.Fatalf("RunWithAddr error = %v, want context is required", err)
	}
}

func TestServeRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	srv, err := New(0)
	if err != nil {
		t.Fatalf("new server for nil-context test: %v", err)
	}
	t.Cleanup(func() {
		srv.Close()
	})
	if err := srv.Serve(nil); err == nil || err.Error() != "context is required" {
		t.Fatalf("Serve error = %v, want context is required", err)
	}
}

func TestServerCloseReleasesListener(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	srv, err := New(0)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	addr := srv.Addr()
	if addr == "" {
		t.Fatal("expected non-empty address")
	}

	srv.Close()

	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen after close: %v", err)
	}
	_ = l.Close()
}

func TestOpenAIOAuthConfigFromEnvReturnsNilWhenUnset(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI", "")

	cfg, err := openAIOAuthConfigFromEnv()
	if err != nil {
		t.Fatalf("openai oauth config from env: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config, got %+v", cfg)
	}
}

func TestOpenAIOAuthConfigFromEnvRejectsPartialConfig(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL", "https://provider.example.com/oauth/authorize")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI", "")

	if _, err := openAIOAuthConfigFromEnv(); err == nil {
		t.Fatal("expected error for partial config")
	}
}

func TestOpenAIOAuthConfigFromEnvBuildsConfigWhenComplete(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL", " https://provider.example.com/oauth/authorize ")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL", " https://provider.example.com/oauth/token ")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID", " client-1 ")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET", " secret-1 ")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI", " https://app.example.com/oauth/callback ")

	cfg, err := openAIOAuthConfigFromEnv()
	if err != nil {
		t.Fatalf("openai oauth config from env: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.AuthorizationURL != "https://provider.example.com/oauth/authorize" {
		t.Fatalf("authorization_url = %q", cfg.AuthorizationURL)
	}
	if cfg.TokenURL != "https://provider.example.com/oauth/token" {
		t.Fatalf("token_url = %q", cfg.TokenURL)
	}
	if cfg.ClientID != "client-1" {
		t.Fatalf("client_id = %q", cfg.ClientID)
	}
	if cfg.ClientSecret != "secret-1" {
		t.Fatalf("client_secret = %q", cfg.ClientSecret)
	}
	if cfg.RedirectURI != "https://app.example.com/oauth/callback" {
		t.Fatalf("redirect_uri = %q", cfg.RedirectURI)
	}
}

func TestAISessionGrantConfigFromEnvReturnsNilWhenUnset(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER", "")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE", "")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY", "")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_TTL", "")

	cfg, err := aiSessionGrantConfigFromEnv()
	if err != nil {
		t.Fatalf("ai session grant config from env: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config, got %+v", cfg)
	}
}

func TestAISessionGrantConfigFromEnvBuildsConfigWhenComplete(t *testing.T) {
	setAISessionGrantEnv(t)

	cfg, err := aiSessionGrantConfigFromEnv()
	if err != nil {
		t.Fatalf("ai session grant config from env: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Issuer != "fracturing-space-game" {
		t.Fatalf("issuer = %q", cfg.Issuer)
	}
	if cfg.Audience != "fracturing-space-ai" {
		t.Fatalf("audience = %q", cfg.Audience)
	}
}
