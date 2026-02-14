package passkey

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/branding"
)

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	cfg := LoadConfigFromEnv()
	if cfg.RPID != defaultRPID {
		t.Fatalf("RPID = %q, want %q", cfg.RPID, defaultRPID)
	}
	if cfg.RPDisplayName != branding.AppName {
		t.Fatalf("RPDisplayName = %q, want %q", cfg.RPDisplayName, branding.AppName)
	}
	if len(cfg.RPOrigins) != 1 || cfg.RPOrigins[0] != defaultOrigin {
		t.Fatalf("RPOrigins = %v, want [%q]", cfg.RPOrigins, defaultOrigin)
	}
	if cfg.SessionTTL != defaultSession {
		t.Fatalf("SessionTTL = %v, want %v", cfg.SessionTTL, defaultSession)
	}
}

func TestLoadConfigFromEnvCustomRPID(t *testing.T) {
	t.Setenv(envRPID, "example.com")
	cfg := LoadConfigFromEnv()
	if cfg.RPID != "example.com" {
		t.Fatalf("RPID = %q, want %q", cfg.RPID, "example.com")
	}
}

func TestLoadConfigFromEnvCustomRPName(t *testing.T) {
	t.Setenv(envRPName, "My App")
	cfg := LoadConfigFromEnv()
	if cfg.RPDisplayName != "My App" {
		t.Fatalf("RPDisplayName = %q, want %q", cfg.RPDisplayName, "My App")
	}
}

func TestLoadConfigFromEnvCustomOrigins(t *testing.T) {
	t.Setenv(envRPOrigins, "https://a.com,https://b.com")
	cfg := LoadConfigFromEnv()
	if len(cfg.RPOrigins) != 2 {
		t.Fatalf("RPOrigins len = %d, want 2", len(cfg.RPOrigins))
	}
	if cfg.RPOrigins[0] != "https://a.com" || cfg.RPOrigins[1] != "https://b.com" {
		t.Fatalf("RPOrigins = %v", cfg.RPOrigins)
	}
}

func TestLoadConfigFromEnvValidSessionTTL(t *testing.T) {
	t.Setenv(envSessionTTL, "10m")
	cfg := LoadConfigFromEnv()
	if cfg.SessionTTL != 10*time.Minute {
		t.Fatalf("SessionTTL = %v, want %v", cfg.SessionTTL, 10*time.Minute)
	}
}

func TestLoadConfigFromEnvInvalidSessionTTLFallsBack(t *testing.T) {
	t.Setenv(envSessionTTL, "bad")
	cfg := LoadConfigFromEnv()
	if cfg.SessionTTL != defaultSession {
		t.Fatalf("SessionTTL = %v, want %v", cfg.SessionTTL, defaultSession)
	}
}

func TestParseCSVEnvEmpty(t *testing.T) {
	result := parseCSVEnv("NONEXISTENT_ENV_VAR_FOR_TEST")
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestParseCSVEnvSingle(t *testing.T) {
	t.Setenv("TEST_CSV_SINGLE", "https://a.com")
	result := parseCSVEnv("TEST_CSV_SINGLE")
	if len(result) != 1 || result[0] != "https://a.com" {
		t.Fatalf("result = %v", result)
	}
}

func TestParseCSVEnvMultiple(t *testing.T) {
	t.Setenv("TEST_CSV_MULTI", "https://a.com, https://b.com , https://c.com")
	result := parseCSVEnv("TEST_CSV_MULTI")
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
	}
	if result[0] != "https://a.com" || result[1] != "https://b.com" || result[2] != "https://c.com" {
		t.Fatalf("result = %v", result)
	}
}

func TestParseCSVEnvEmptyEntriesTrimmed(t *testing.T) {
	t.Setenv("TEST_CSV_EMPTY", "https://a.com,,  , https://b.com")
	result := parseCSVEnv("TEST_CSV_EMPTY")
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0] != "https://a.com" || result[1] != "https://b.com" {
		t.Fatalf("result = %v", result)
	}
}
