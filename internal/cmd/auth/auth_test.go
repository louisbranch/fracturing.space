package auth

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("auth", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 8083 {
		t.Fatalf("expected default port 8083, got %d", cfg.Port)
	}
	if cfg.HTTPAddr != "localhost:8084" {
		t.Fatalf("expected default http addr, got %q", cfg.HTTPAddr)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	fs := flag.NewFlagSet("auth", flag.ContinueOnError)
	lookup := func(key string) (string, bool) {
		if key == "FRACTURING_SPACE_AUTH_HTTP_ADDR" {
			return "env-http", true
		}
		return "", false
	}
	args := []string{"-port", "9000", "-http-addr", "flag-http"}
	cfg, err := ParseConfig(fs, args, lookup)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 9000 {
		t.Fatalf("expected port 9000, got %d", cfg.Port)
	}
	if cfg.HTTPAddr != "flag-http" {
		t.Fatalf("expected flag http addr, got %q", cfg.HTTPAddr)
	}
}
