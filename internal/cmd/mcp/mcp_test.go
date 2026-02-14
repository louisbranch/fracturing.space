package mcp

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Addr != "localhost:8080" {
		t.Fatalf("expected default addr, got %q", cfg.Addr)
	}
	if cfg.HTTPAddr != "localhost:8081" {
		t.Fatalf("expected default http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.Transport != "stdio" {
		t.Fatalf("expected default transport stdio, got %q", cfg.Transport)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_ADDR", "env-game")
	t.Setenv("FRACTURING_SPACE_MCP_HTTP_ADDR", "env-http")

	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	args := []string{"-addr", "flag-game", "-http-addr", "flag-http", "-transport", "http"}
	cfg, err := ParseConfig(fs, args)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Addr != "flag-game" {
		t.Fatalf("expected flag addr, got %q", cfg.Addr)
	}
	if cfg.HTTPAddr != "flag-http" {
		t.Fatalf("expected flag http addr, got %q", cfg.HTTPAddr)
	}
	if cfg.Transport != "http" {
		t.Fatalf("expected transport http, got %q", cfg.Transport)
	}
}
