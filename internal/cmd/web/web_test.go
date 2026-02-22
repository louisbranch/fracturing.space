package web

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.HTTPAddr != "localhost:8080" {
		t.Fatalf("expected default http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.ChatHTTPAddr != "localhost:8086" {
		t.Fatalf("expected default chat http addr, got %q", cfg.ChatHTTPAddr)
	}
	if cfg.AuthBaseURL != "http://localhost:8084" {
		t.Fatalf("expected default auth base url, got %q", cfg.AuthBaseURL)
	}
	if cfg.AuthAddr != "localhost:8083" {
		t.Fatalf("expected default auth addr, got %q", cfg.AuthAddr)
	}
	if cfg.OAuthClientID != "fracturing-space" {
		t.Fatalf("expected default oauth client id, got %q", cfg.OAuthClientID)
	}
	if cfg.GameAddr != "localhost:8082" {
		t.Fatalf("expected default game addr, got %q", cfg.GameAddr)
	}
	if cfg.NotificationsAddr != "localhost:8088" {
		t.Fatalf("expected default notifications addr, got %q", cfg.NotificationsAddr)
	}
	if cfg.AIAddr != "" {
		t.Fatalf("expected default ai addr to be empty, got %q", cfg.AIAddr)
	}
	if cfg.CacheDBPath != "data/web-cache.db" {
		t.Fatalf("expected default cache db path, got %q", cfg.CacheDBPath)
	}
	if cfg.AssetVersion != "v1" {
		t.Fatalf("expected default asset version, got %q", cfg.AssetVersion)
	}
	if cfg.AssetBaseURL != "" {
		t.Fatalf("expected empty default asset base url, got %q", cfg.AssetBaseURL)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{
		"-http-addr", "127.0.0.1:9999",
		"-chat-http-addr", "chat.internal:10001",
		"-auth-base-url", "http://auth.test",
		"-auth-addr", "auth:9000",
		"-game-addr", "game:9001",
		"-notifications-addr", "notifications:9003",
		"-ai-addr", "ai:9002",
		"-cache-db-path", "/tmp/web-cache.db",
		"-asset-base-url", "https://assets.test",
		"-asset-version", "v9",
	})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:9999" {
		t.Fatalf("expected overridden http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.ChatHTTPAddr != "chat.internal:10001" {
		t.Fatalf("expected overridden chat http addr, got %q", cfg.ChatHTTPAddr)
	}
	if cfg.AuthBaseURL != "http://auth.test" {
		t.Fatalf("expected overridden auth base url, got %q", cfg.AuthBaseURL)
	}
	if cfg.AuthAddr != "auth:9000" {
		t.Fatalf("expected overridden auth addr, got %q", cfg.AuthAddr)
	}
	if cfg.GameAddr != "game:9001" {
		t.Fatalf("expected overridden game addr, got %q", cfg.GameAddr)
	}
	if cfg.NotificationsAddr != "notifications:9003" {
		t.Fatalf("expected overridden notifications addr, got %q", cfg.NotificationsAddr)
	}
	if cfg.AIAddr != "ai:9002" {
		t.Fatalf("expected overridden ai addr, got %q", cfg.AIAddr)
	}
	if cfg.CacheDBPath != "/tmp/web-cache.db" {
		t.Fatalf("expected overridden cache db path, got %q", cfg.CacheDBPath)
	}
	if cfg.AssetBaseURL != "https://assets.test" {
		t.Fatalf("expected overridden asset base url, got %q", cfg.AssetBaseURL)
	}
	if cfg.AssetVersion != "v9" {
		t.Fatalf("expected overridden asset version, got %q", cfg.AssetVersion)
	}
}
