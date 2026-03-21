// Package ai parses AI command flags and launches the AI control-plane service.
package ai

import (
	"context"
	"flag"
	"fmt"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	server "github.com/louisbranch/fracturing.space/internal/services/ai/app"
)

// Config holds AI command configuration.
type Config struct {
	Port       int    `env:"FRACTURING_SPACE_AI_PORT" envDefault:"8087"`
	StatusAddr string `env:"FRACTURING_SPACE_STATUS_ADDR"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The AI gRPC server port")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the AI orchestration service.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceAI, func(context.Context) error {
		stopReporter := entrypoint.StartStatusReporter(
			ctx,
			"ai",
			cfg.StatusAddr,
			entrypoint.Capability("ai.credentials", platformstatus.Operational),
			entrypoint.Capability("ai.agents", platformstatus.Operational),
		)
		defer stopReporter()

		return server.Run(ctx, fmt.Sprintf(":%d", cfg.Port))
	})
}
