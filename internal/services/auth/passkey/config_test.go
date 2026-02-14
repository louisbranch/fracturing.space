package passkey

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/branding"
)

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	cfg := LoadConfigFromEnv()
	if cfg.RPID != "localhost" {
		t.Fatalf("RPID = %q, want %q", cfg.RPID, "localhost")
	}
	if cfg.RPDisplayName != branding.AppName {
		t.Fatalf("RPDisplayName = %q, want %q", cfg.RPDisplayName, branding.AppName)
	}
	if len(cfg.RPOrigins) != 1 || cfg.RPOrigins[0] != "http://localhost:8086" {
		t.Fatalf("RPOrigins = %v, want [%q]", cfg.RPOrigins, "http://localhost:8086")
	}
	if cfg.SessionTTL != 5*time.Minute {
		t.Fatalf("SessionTTL = %v, want %v", cfg.SessionTTL, 5*time.Minute)
	}
}

func TestLoadConfigFromEnvCustomRPID(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_WEBAUTHN_RP_ID", "example.com")
	cfg := LoadConfigFromEnv()
	if cfg.RPID != "example.com" {
		t.Fatalf("RPID = %q, want %q", cfg.RPID, "example.com")
	}
}

func TestLoadConfigFromEnvCustomRPName(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_WEBAUTHN_RP_DISPLAY_NAME", "My App")
	cfg := LoadConfigFromEnv()
	if cfg.RPDisplayName != "My App" {
		t.Fatalf("RPDisplayName = %q, want %q", cfg.RPDisplayName, "My App")
	}
}

func TestLoadConfigFromEnvCustomOrigins(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_WEBAUTHN_RP_ORIGINS", "https://a.com,https://b.com")
	cfg := LoadConfigFromEnv()
	if len(cfg.RPOrigins) != 2 {
		t.Fatalf("RPOrigins len = %d, want 2", len(cfg.RPOrigins))
	}
	if cfg.RPOrigins[0] != "https://a.com" || cfg.RPOrigins[1] != "https://b.com" {
		t.Fatalf("RPOrigins = %v", cfg.RPOrigins)
	}
}

func TestLoadConfigFromEnvValidSessionTTL(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_WEBAUTHN_SESSION_TTL", "10m")
	cfg := LoadConfigFromEnv()
	if cfg.SessionTTL != 10*time.Minute {
		t.Fatalf("SessionTTL = %v, want %v", cfg.SessionTTL, 10*time.Minute)
	}
}

func TestLoadConfigFromEnvInvalidSessionTTLKeepsRPID(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_WEBAUTHN_RP_ID", "example.com")
	t.Setenv("FRACTURING_SPACE_WEBAUTHN_SESSION_TTL", "bad-duration")

	cfg := LoadConfigFromEnv()
	if cfg.RPID != "example.com" {
		t.Fatalf("RPID = %q, want %q", cfg.RPID, "example.com")
	}
	if cfg.SessionTTL != 5*time.Minute {
		t.Fatalf("SessionTTL = %v, want %v", cfg.SessionTTL, 5*time.Minute)
	}
}
