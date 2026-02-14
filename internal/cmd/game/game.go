package game

import (
	"context"
	"flag"

	server "github.com/louisbranch/fracturing.space/internal/services/game/app"
)

// Config holds game command configuration.
type Config struct {
	Port int
	Addr string
}

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	cfg := Config{Port: 8080}
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The game server port")
	fs.StringVar(&cfg.Addr, "addr", "", "The game server listen address (overrides -port)")
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
