// Package status parses status service flags and launches the service.
package status

import (
	"context"
	"flag"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	server "github.com/louisbranch/fracturing.space/internal/services/status/app"
)

// Config holds status command configuration.
type Config struct {
	Port int    `env:"FRACTURING_SPACE_STATUS_PORT" envDefault:"8093"`
	Addr string `env:"FRACTURING_SPACE_STATUS_ADDR"`
}

// ParseConfig parses environment and flags into Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The status gRPC server port")
	fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "The status server listen address (overrides -port)")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the status gRPC API service.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceStatus, func(context.Context) error {
		if cfg.Addr != "" {
			return server.RunWithAddr(ctx, cfg.Addr)
		}
		return server.Run(ctx, cfg.Port)
	})
}
