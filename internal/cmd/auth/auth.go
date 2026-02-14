package auth

import (
	"context"
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
	server "github.com/louisbranch/fracturing.space/internal/services/auth/app"
)

// Config holds auth command configuration.
type Config struct {
	Port     int    `env:"FRACTURING_SPACE_AUTH_PORT"      envDefault:"8083"`
	HTTPAddr string `env:"FRACTURING_SPACE_AUTH_HTTP_ADDR" envDefault:"localhost:8084"`
}

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
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
