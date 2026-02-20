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
	if cfg.HTTPAddr != "localhost:8086" {
		t.Fatalf("expected default http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.AuthBaseURL != "http://localhost:8084" {
		t.Fatalf("expected default auth base url, got %q", cfg.AuthBaseURL)
	}
	if cfg.AuthAddr != "localhost:8083" {
		t.Fatalf("expected default auth addr, got %q", cfg.AuthAddr)
	}
	if cfg.OAuthClientID != "fracturing-space-web" {
		t.Fatalf("expected default oauth client id, got %q", cfg.OAuthClientID)
	}
	if cfg.GameAddr != "localhost:8080" {
		t.Fatalf("expected default game addr, got %q", cfg.GameAddr)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-http-addr", "127.0.0.1:9999", "-auth-base-url", "http://auth.test", "-auth-addr", "auth:9000", "-game-addr", "game:9001"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:9999" {
		t.Fatalf("expected overridden http addr, got %q", cfg.HTTPAddr)
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
}
