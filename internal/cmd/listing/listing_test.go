package listing

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("listing", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 8091 {
		t.Fatalf("expected default port 8091, got %d", cfg.Port)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_LISTING_PORT", "9090")

	fs := flag.NewFlagSet("listing", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-port", "9091"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 9091 {
		t.Fatalf("expected port override 9091, got %d", cfg.Port)
	}
}
