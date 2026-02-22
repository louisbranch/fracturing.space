// Package game parses game command flags and starts the domain runtime.
package game

import (
	"context"
	"flag"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	server "github.com/louisbranch/fracturing.space/internal/services/game/app"
)

// Config holds game command configuration.
type Config struct {
	Port int    `env:"FRACTURING_SPACE_GAME_PORT" envDefault:"8082"`
	Addr string `env:"FRACTURING_SPACE_GAME_ADDR"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The game server port")
	fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "The game server listen address (overrides -port)")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the game domain API service.
func Run(ctx context.Context, cfg Config) error {
	if err := catalog.ValidateEmbeddedCatalogManifests(); err != nil {
		return err
	}
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceGame, func(context.Context) error {
		if cfg.Addr != "" {
			return server.RunWithAddr(ctx, cfg.Addr)
		}
		return server.Run(ctx, cfg.Port)
	})
}
