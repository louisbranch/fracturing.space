package auth

import (
	"context"
	"flag"
	"strings"

	server "github.com/louisbranch/fracturing.space/internal/services/auth/app"
)

// Config holds auth command configuration.
type Config struct {
	Port     int
	HTTPAddr string
}

// EnvLookup returns the value for a key when present.
type EnvLookup func(string) (string, bool)

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string, lookup EnvLookup) (Config, error) {
	cfg := Config{
		Port:     8083,
		HTTPAddr: envOrDefault(lookup, []string{"FRACTURING_SPACE_AUTH_HTTP_ADDR"}, "localhost:8084"),
	}

	fs.IntVar(&cfg.Port, "port", cfg.Port, "The auth gRPC server port")
	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "The auth HTTP server address")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the auth server.
func Run(ctx context.Context, cfg Config) error {
	return server.Run(ctx, cfg.Port, cfg.HTTPAddr)
}

func envOrDefault(lookup EnvLookup, keys []string, fallback string) string {
	for _, key := range keys {
		if lookup == nil {
			break
		}
		value, ok := lookup(key)
		if ok {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}
