package server

import (
	"encoding/base64"
	"net"
	"path/filepath"
	"testing"
)

func TestNewRequiresEncryptionKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", "")

	if _, err := New(0); err == nil {
		t.Fatal("expected error for missing encryption key")
	}
}

func TestNewRejectsInvalidEncryptionKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", "not-base64")

	if _, err := New(0); err == nil {
		t.Fatal("expected error for invalid encryption key")
	}
}

func TestNewSuccess(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))

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

func TestServerCloseReleasesListener(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))

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
