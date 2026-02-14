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
	if err := env.Parse(&cfg); err != nil {
		return Config{
			RPDisplayName: branding.AppName,
			RPID:          "localhost",
			RPOrigins:     []string{"http://localhost:8086"},
			SessionTTL:    5 * time.Minute,
		}
	}
	if cfg.RPDisplayName == "" {
		cfg.RPDisplayName = branding.AppName
	}
	if len(cfg.RPOrigins) == 0 {
		cfg.RPOrigins = []string{"http://localhost:8086"}
	}
	return cfg
}
