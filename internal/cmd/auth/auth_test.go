package auth

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("auth", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
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
	t.Setenv("FRACTURING_SPACE_AUTH_HTTP_ADDR", "env-http")

	fs := flag.NewFlagSet("auth", flag.ContinueOnError)
	args := []string{"-port", "9000", "-http-addr", "flag-http"}
	cfg, err := ParseConfig(fs, args)
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
