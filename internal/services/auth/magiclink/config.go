package magiclink

import (
	"time"

	"github.com/caarlos0/env/v11"
)

// Config controls magic link timing and redirect behavior for authentication.
//
// These values are read at startup so operator-controlled defaults can be tuned
// without changing runtime code paths.
type Config struct {
	BaseURL string        `env:"FRACTURING_SPACE_MAGIC_LINK_BASE_URL" envDefault:"http://localhost:8086/magic"`
	TTL     time.Duration `env:"FRACTURING_SPACE_MAGIC_LINK_TTL"      envDefault:"15m"`
}

// LoadConfigFromEnv loads magic-link configuration and applies defensive defaults.
//
// Defaults are intentionally explicit because magic links are security-sensitive and
// should remain predictable in local and CI environments.
func LoadConfigFromEnv() Config {
	var cfg Config
	_ = env.Parse(&cfg)
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8086/magic"
	}
	if cfg.TTL == 0 {
		cfg.TTL = 15 * time.Minute
	}
	return cfg
}
