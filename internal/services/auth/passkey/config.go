package passkey

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
)

// SessionKind describes the WebAuthn session purpose.
type SessionKind string

const (
	SessionKindRegistration SessionKind = "registration"
	SessionKindLogin        SessionKind = "login"
)

// Config controls WebAuthn relying party settings.
type Config struct {
	RPDisplayName string        `env:"FRACTURING_SPACE_WEBAUTHN_RP_DISPLAY_NAME"`
	RPID          string        `env:"FRACTURING_SPACE_WEBAUTHN_RP_ID"           envDefault:"localhost"`
	RPOrigins     []string      `env:"FRACTURING_SPACE_WEBAUTHN_RP_ORIGINS"      envSeparator:","`
	SessionTTL    time.Duration `env:"FRACTURING_SPACE_WEBAUTHN_SESSION_TTL"     envDefault:"5m"`
}

// LoadConfigFromEnv returns passkey configuration with defaults.
func LoadConfigFromEnv() Config {
	var cfg Config
	_ = env.Parse(&cfg)
	if cfg.RPDisplayName == "" {
		cfg.RPDisplayName = branding.AppName
	}
	if cfg.RPID == "" {
		cfg.RPID = "localhost"
	}
	if len(cfg.RPOrigins) == 0 {
		cfg.RPOrigins = []string{"http://localhost:8086"}
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = 5 * time.Minute
	}
	return cfg
}
