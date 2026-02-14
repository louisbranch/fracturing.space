package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"
)

func TestDecodeBase64(t *testing.T) {
	t.Run("empty value", func(t *testing.T) {
		_, err := decodeBase64("")
		if err == nil {
			t.Fatal("expected error for empty value")
		}
	})

	t.Run("raw std encoding", func(t *testing.T) {
		input := base64.RawStdEncoding.EncodeToString([]byte("hello"))
		got, err := decodeBase64(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "hello" {
			t.Errorf("expected %q, got %q", "hello", string(got))
		}
	})

	t.Run("std encoding with padding", func(t *testing.T) {
		input := base64.StdEncoding.EncodeToString([]byte("hello"))
		got, err := decodeBase64(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "hello" {
			t.Errorf("expected %q, got %q", "hello", string(got))
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := decodeBase64("!!!not-base64!!!")
		if err == nil {
			t.Fatal("expected error for invalid base64")
		}
	})
}

func TestLoadJoinGrantConfigFromEnv(t *testing.T) {
	t.Run("missing issuer", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "")
		_, err := loadJoinGrantConfigFromEnv()
		if err == nil {
			t.Fatal("expected error for missing issuer")
		}
	})

	t.Run("issuer whitespace", func(t *testing.T) {
		_, key, _ := ed25519.GenerateKey(nil)
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "  ")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "test-audience")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", base64.StdEncoding.EncodeToString(key))
		_, err := loadJoinGrantConfigFromEnv()
		if err == nil {
			t.Fatal("expected error for whitespace issuer")
		}
	})

	t.Run("missing audience", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "test-issuer")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "")
		_, err := loadJoinGrantConfigFromEnv()
		if err == nil {
			t.Fatal("expected error for missing audience")
		}
	})

	t.Run("missing private key", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "test-issuer")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "test-audience")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", "")
		_, err := loadJoinGrantConfigFromEnv()
		if err == nil {
			t.Fatal("expected error for missing private key")
		}
	})

	t.Run("invalid base64 key", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "test-issuer")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "test-audience")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", "!!!not-base64!!!")
		_, err := loadJoinGrantConfigFromEnv()
		if err == nil {
			t.Fatal("expected error for invalid base64 key")
		}
	})

	t.Run("wrong key size", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "test-issuer")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "test-audience")
		// 16 bytes instead of 64
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", base64.StdEncoding.EncodeToString(make([]byte, 16)))
		_, err := loadJoinGrantConfigFromEnv()
		if err == nil {
			t.Fatal("expected error for wrong key size")
		}
	})

	t.Run("invalid TTL", func(t *testing.T) {
		_, key, _ := ed25519.GenerateKey(nil)
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "test-issuer")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "test-audience")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", base64.StdEncoding.EncodeToString(key))
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_TTL", "not-a-duration")
		_, err := loadJoinGrantConfigFromEnv()
		if err == nil {
			t.Fatal("expected error for invalid TTL")
		}
	})

	t.Run("negative TTL", func(t *testing.T) {
		_, key, _ := ed25519.GenerateKey(nil)
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "test-issuer")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "test-audience")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", base64.StdEncoding.EncodeToString(key))
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_TTL", "-5m")
		_, err := loadJoinGrantConfigFromEnv()
		if err == nil {
			t.Fatal("expected error for negative TTL")
		}
	})

	t.Run("success with default TTL", func(t *testing.T) {
		_, key, _ := ed25519.GenerateKey(nil)
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "test-issuer")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "test-audience")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", base64.StdEncoding.EncodeToString(key))
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_TTL", "")
		cfg, err := loadJoinGrantConfigFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.issuer != "test-issuer" {
			t.Errorf("issuer = %q, want %q", cfg.issuer, "test-issuer")
		}
		if cfg.audience != "test-audience" {
			t.Errorf("audience = %q, want %q", cfg.audience, "test-audience")
		}
		if cfg.ttl != 5*time.Minute {
			t.Errorf("ttl = %v, want %v", cfg.ttl, 5*time.Minute)
		}
	})

	t.Run("success with custom TTL", func(t *testing.T) {
		_, key, _ := ed25519.GenerateKey(nil)
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", "test-issuer")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", "test-audience")
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", base64.StdEncoding.EncodeToString(key))
		t.Setenv("FRACTURING_SPACE_JOIN_GRANT_TTL", "10m")
		cfg, err := loadJoinGrantConfigFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ttl.Minutes() != 10 {
			t.Errorf("ttl = %v, want 10m", cfg.ttl)
		}
	})
}

func TestEncodeJoinGrant(t *testing.T) {
	t.Run("unconfigured signer", func(t *testing.T) {
		_, err := encodeJoinGrant(joinGrantConfig{}, nil)
		if err == nil {
			t.Fatal("expected error for unconfigured signer")
		}
	})

	t.Run("success", func(t *testing.T) {
		_, key, _ := ed25519.GenerateKey(nil)
		cfg := joinGrantConfig{
			issuer:   "test-issuer",
			audience: "test-audience",
			key:      key,
		}
		token, err := encodeJoinGrant(cfg, map[string]any{
			"sub":  "user-1",
			"role": "player",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token == "" {
			t.Fatal("expected non-empty token")
		}
		// JWT has 3 parts separated by dots
		parts := 0
		for _, c := range token {
			if c == '.' {
				parts++
			}
		}
		if parts != 2 {
			t.Errorf("expected 3 JWT segments (2 dots), got %d dots", parts)
		}
	})
}
