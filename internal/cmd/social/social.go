// Package social parses social service flags and launches the service.
package social

import (
	"context"
	"flag"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	server "github.com/louisbranch/fracturing.space/internal/services/social/app"
)

// Config holds social command configuration.
type Config struct {
	Port int `env:"FRACTURING_SPACE_SOCIAL_PORT" envDefault:"8090"`
}

// ParseConfig parses environment and flags into Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The social gRPC server port")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the social gRPC API service.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceSocial, func(context.Context) error {
		return server.Run(ctx, cfg.Port)
	})
}
