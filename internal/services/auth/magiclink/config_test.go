package magiclink

import (
	"testing"
	"time"
)

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	cfg := LoadConfigFromEnv()
	if cfg.BaseURL != defaultBase {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, defaultBase)
	}
	if cfg.TTL != defaultTTL {
		t.Fatalf("TTL = %v, want %v", cfg.TTL, defaultTTL)
	}
}

func TestLoadConfigFromEnvCustomBaseURL(t *testing.T) {
	t.Setenv(envBaseURL, "https://example.com/magic")
	cfg := LoadConfigFromEnv()
	if cfg.BaseURL != "https://example.com/magic" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "https://example.com/magic")
	}
}

func TestLoadConfigFromEnvValidTTL(t *testing.T) {
	t.Setenv(envTTL, "30m")
	cfg := LoadConfigFromEnv()
	if cfg.TTL != 30*time.Minute {
		t.Fatalf("TTL = %v, want %v", cfg.TTL, 30*time.Minute)
	}
}

func TestLoadConfigFromEnvInvalidTTLFallsBack(t *testing.T) {
	t.Setenv(envTTL, "not-a-duration")
	cfg := LoadConfigFromEnv()
	if cfg.TTL != defaultTTL {
		t.Fatalf("TTL = %v, want %v", cfg.TTL, defaultTTL)
	}
}
