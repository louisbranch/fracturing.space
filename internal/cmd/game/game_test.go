package game

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("game", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.Addr != "" {
		t.Fatalf("expected empty addr, got %q", cfg.Addr)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	fs := flag.NewFlagSet("game", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-port", "9001", "-addr", "127.0.0.1:9999"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 9001 {
		t.Fatalf("expected port 9001, got %d", cfg.Port)
	}
	if cfg.Addr != "127.0.0.1:9999" {
		t.Fatalf("expected addr override, got %q", cfg.Addr)
	}
}
