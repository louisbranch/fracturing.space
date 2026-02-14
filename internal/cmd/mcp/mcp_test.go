package mcp

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil, func(string) (string, bool) { return "", false })
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
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	lookup := func(key string) (string, bool) {
		switch key {
		case "FRACTURING_SPACE_GAME_ADDR":
			return "env-game", true
		case "FRACTURING_SPACE_MCP_HTTP_ADDR":
			return "env-http", true
		default:
			return "", false
		}
	}
	args := []string{"-addr", "flag-game", "-http-addr", "flag-http", "-transport", "http"}
	cfg, err := ParseConfig(fs, args, lookup)
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
