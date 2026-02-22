// Package notifications parses notifications command flags and launches the service.
package notifications

import (
	"context"
	"flag"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	server "github.com/louisbranch/fracturing.space/internal/services/notifications/app"
)

// Config holds notifications command configuration.
type Config struct {
	Port int `env:"FRACTURING_SPACE_NOTIFICATIONS_PORT" envDefault:"8088"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The notifications gRPC server port")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the notifications API service.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceNotifications, func(context.Context) error {
		return server.Run(ctx, cfg.Port)
	})
}
