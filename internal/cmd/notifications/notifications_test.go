package notifications

import (
	"flag"
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("notifications", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 8088 {
		t.Fatalf("expected default port 8088, got %d", cfg.Port)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_NOTIFICATIONS_PORT", "9090")

	fs := flag.NewFlagSet("notifications", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-port", "9091"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 9091 {
		t.Fatalf("expected port override 9091, got %d", cfg.Port)
	}
}
