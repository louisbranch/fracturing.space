// Package ai parses AI command flags and launches the AI control-plane service.
package ai

import (
	"context"
	"flag"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	server "github.com/louisbranch/fracturing.space/internal/services/ai/app"
)

// Config holds AI command configuration.
type Config struct {
	Port int `env:"FRACTURING_SPACE_AI_PORT" envDefault:"8088"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := config.ParseEnv(&cfg); err != nil {
		return Config{}, err
	}
	fs.IntVar(&cfg.Port, "port", cfg.Port, "The AI gRPC server port")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the AI orchestration service.
func Run(ctx context.Context, cfg Config) error {
	return server.Run(ctx, cfg.Port)
}
