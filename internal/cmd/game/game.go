package game

import (
	"context"
	"flag"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	server "github.com/louisbranch/fracturing.space/internal/services/game/app"
)

// Config holds game command configuration.
type Config struct {
	Port int    `env:"FRACTURING_SPACE_GAME_PORT" envDefault:"8080"`
	Addr string `env:"FRACTURING_SPACE_GAME_ADDR"`
}

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := config.ParseEnv(&cfg); err != nil {
		return Config{}, err
	}

	fs.IntVar(&cfg.Port, "port", cfg.Port, "The game server port")
	fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "The game server listen address (overrides -port)")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the game server.
func Run(ctx context.Context, cfg Config) error {
	if cfg.Addr != "" {
		return server.RunWithAddr(ctx, cfg.Addr)
	}
	return server.Run(ctx, cfg.Port)
}
