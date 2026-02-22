package ai

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("ai", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 8087 {
		t.Fatalf("expected default port 8087, got %d", cfg.Port)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_PORT", "9002")

	fs := flag.NewFlagSet("ai", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-port", "9010"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 9010 {
		t.Fatalf("expected port override 9010, got %d", cfg.Port)
	}
}
