// Package social parses social service flags and launches the service.
package social

import (
	"context"
	"flag"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	server "github.com/louisbranch/fracturing.space/internal/services/social/app"
)

// Config holds social command configuration.
type Config struct {
	Port       int    `env:"FRACTURING_SPACE_SOCIAL_PORT" envDefault:"8090"`
	StatusAddr string `env:"FRACTURING_SPACE_STATUS_ADDR"`
}

// ParseConfig parses environment and flags into Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The social gRPC server port")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the social gRPC API service.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceSocial, func(context.Context) error {
		stopReporter := entrypoint.StartStatusReporter(
			ctx,
			"social",
			cfg.StatusAddr,
			entrypoint.Capability("social.profiles", platformstatus.Operational),
		)
		defer stopReporter()

		return server.Run(ctx, cfg.Port)
	})
}
