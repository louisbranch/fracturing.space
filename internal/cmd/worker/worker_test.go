package worker

import (
	"flag"
	"testing"
)

func TestParseConfig_ParsesDefaultsAndFlags(t *testing.T) {
	fs := flag.NewFlagSet("worker", flag.ContinueOnError)
	t.Setenv("FRACTURING_SPACE_WORKER_PORT", "9099")
	t.Setenv("FRACTURING_SPACE_WORKER_AUTH_ADDR", "auth:8083")

	cfg, err := ParseConfig(fs, []string{"-consumer", "worker-e2e", "-max-attempts", "3"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Port != 9099 {
		t.Fatalf("port = %d, want 9099", cfg.Port)
	}
	if cfg.AuthAddr != "auth:8083" {
		t.Fatalf("auth addr = %q, want %q", cfg.AuthAddr, "auth:8083")
	}
	if cfg.Consumer != "worker-e2e" {
		t.Fatalf("consumer = %q, want %q", cfg.Consumer, "worker-e2e")
	}
	if cfg.MaxAttempts != 3 {
		t.Fatalf("max attempts = %d, want 3", cfg.MaxAttempts)
	}
}

func TestParseConfig_DefaultDiscoveryAddresses(t *testing.T) {
	fs := flag.NewFlagSet("worker", flag.ContinueOnError)

	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.AuthAddr != "auth:8083" {
		t.Fatalf("auth addr = %q, want %q", cfg.AuthAddr, "auth:8083")
	}
	if cfg.SocialAddr != "social:8090" {
		t.Fatalf("social addr = %q, want %q", cfg.SocialAddr, "social:8090")
	}
	if cfg.NotificationsAddr != "notifications:8088" {
		t.Fatalf("notifications addr = %q, want %q", cfg.NotificationsAddr, "notifications:8088")
	}
}
