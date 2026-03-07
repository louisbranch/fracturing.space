// Package auth parses auth service startup flags and hands off to the app server.
package auth

import (
	"context"
	"flag"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	server "github.com/louisbranch/fracturing.space/internal/services/auth/app"
)

// Config holds auth command configuration.
type Config struct {
	Port       int    `env:"FRACTURING_SPACE_AUTH_PORT"      envDefault:"8083"`
	HTTPAddr   string `env:"FRACTURING_SPACE_AUTH_HTTP_ADDR" envDefault:"localhost:8084"`
	StatusAddr string `env:"FRACTURING_SPACE_STATUS_ADDR"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)

	fs.IntVar(&cfg.Port, "port", cfg.Port, "The auth gRPC server port")
	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "The auth HTTP server address")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the auth process by delegating to the auth app server.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceAuth, func(context.Context) error {
		stopReporter := entrypoint.StartStatusReporter(
			ctx,
			"auth",
			cfg.StatusAddr,
			entrypoint.Capability("auth.identity", platformstatus.Operational),
			entrypoint.Capability("auth.oauth", platformstatus.Operational),
		)
		defer stopReporter()

		return server.Run(ctx, cfg.Port, cfg.HTTPAddr)
	})
}
