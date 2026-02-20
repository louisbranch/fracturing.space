// Package game parses game command flags and starts the domain runtime.
package game

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/platform/otel"
	server "github.com/louisbranch/fracturing.space/internal/services/game/app"
)

// Config holds game command configuration.
type Config struct {
	Port int    `env:"FRACTURING_SPACE_GAME_PORT" envDefault:"8080"`
	Addr string `env:"FRACTURING_SPACE_GAME_ADDR"`
}

// ParseConfig parses environment and flags into a Config.
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

// Run starts the game domain API service.
func Run(ctx context.Context, cfg Config) error {
	shutdown, err := otel.Setup(ctx, "game")
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			log.Printf("otel shutdown: %v", err)
		}
	}()

	if cfg.Addr != "" {
		return server.RunWithAddr(ctx, cfg.Addr)
	}
	return server.Run(ctx, cfg.Port)
}
