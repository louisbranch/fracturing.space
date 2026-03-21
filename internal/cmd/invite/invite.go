// Package invite parses invite service flags and launches the service.
package invite

import (
	"context"
	"flag"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	server "github.com/louisbranch/fracturing.space/internal/services/invite/app"
)

// Config holds invite command configuration.
type Config struct {
	Port     int    `env:"FRACTURING_SPACE_INVITE_PORT" envDefault:"8095"`
	GameAddr string `env:"FRACTURING_SPACE_INVITE_GAME_ADDR"`
	AuthAddr string `env:"FRACTURING_SPACE_INVITE_AUTH_ADDR"`
}

// ParseConfig parses environment and flags into Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.GameAddr = serviceaddr.OrDefaultGRPCAddr(cfg.GameAddr, serviceaddr.ServiceGame)
	cfg.AuthAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AuthAddr, serviceaddr.ServiceAuth)
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The invite gRPC server port")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the invite gRPC API service.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceInvite, func(context.Context) error {
		return server.Run(ctx, server.Config{
			Port:     cfg.Port,
			GameAddr: cfg.GameAddr,
			AuthAddr: cfg.AuthAddr,
		})
	})
}
