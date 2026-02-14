package magiclink

import (
	"os"
	"strings"
	"time"
)

const (
	envBaseURL  = "FRACTURING_SPACE_MAGIC_LINK_BASE_URL"
	envTTL      = "FRACTURING_SPACE_MAGIC_LINK_TTL"
	defaultBase = "http://localhost:8086/magic"
	defaultTTL  = 15 * time.Minute
)

// Config controls magic link generation.
type Config struct {
	BaseURL string
	TTL     time.Duration
}

// LoadConfigFromEnv loads config from environment variables.
func LoadConfigFromEnv() Config {
	base := strings.TrimSpace(os.Getenv(envBaseURL))
	if base == "" {
		base = defaultBase
	}

	ttl := defaultTTL
	if raw := strings.TrimSpace(os.Getenv(envTTL)); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			ttl = parsed
		}
	}

	return Config{BaseURL: base, TTL: ttl}
}
