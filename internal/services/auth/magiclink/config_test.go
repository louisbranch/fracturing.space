package magiclink

import (
	"testing"
	"time"
)

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	cfg := LoadConfigFromEnv()
	if cfg.BaseURL != "http://localhost:8086/magic" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:8086/magic")
	}
	if cfg.TTL != 15*time.Minute {
		t.Fatalf("TTL = %v, want %v", cfg.TTL, 15*time.Minute)
	}
}

func TestLoadConfigFromEnvCustomBaseURL(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_MAGIC_LINK_BASE_URL", "https://example.com/magic")
	cfg := LoadConfigFromEnv()
	if cfg.BaseURL != "https://example.com/magic" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "https://example.com/magic")
	}
}

func TestLoadConfigFromEnvValidTTL(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_MAGIC_LINK_TTL", "30m")
	cfg := LoadConfigFromEnv()
	if cfg.TTL != 30*time.Minute {
		t.Fatalf("TTL = %v, want %v", cfg.TTL, 30*time.Minute)
	}
}
