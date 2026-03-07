// Package discovery parses discovery service flags and launches the service.
package discovery

import (
	"context"
	"flag"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	server "github.com/louisbranch/fracturing.space/internal/services/discovery/app"
)

// Config holds discovery command configuration.
type Config struct {
	Port       int    `env:"FRACTURING_SPACE_DISCOVERY_PORT" envDefault:"8091"`
	StatusAddr string `env:"FRACTURING_SPACE_STATUS_ADDR"`
}

// ParseConfig parses environment and flags into Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The discovery gRPC server port")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the discovery gRPC API service.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceDiscovery, func(context.Context) error {
		stopReporter := entrypoint.StartStatusReporter(
			ctx,
			"discovery",
			cfg.StatusAddr,
			entrypoint.Capability("discovery.entries", platformstatus.Operational),
		)
		defer stopReporter()

		return server.Run(ctx, cfg.Port)
	})
}
