package mcp

import (
	"context"
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
	mcpapp "github.com/louisbranch/fracturing.space/internal/services/mcp/app"
)

// Config holds MCP command configuration.
type Config struct {
	Addr      string `env:"FRACTURING_SPACE_GAME_ADDR"     envDefault:"localhost:8080"`
	HTTPAddr  string `env:"FRACTURING_SPACE_MCP_HTTP_ADDR" envDefault:"localhost:8081"`
	Transport string `env:"FRACTURING_SPACE_MCP_TRANSPORT" envDefault:"stdio"`
}

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}

	fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "game server address")
	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP server address (for HTTP transport)")
	fs.StringVar(&cfg.Transport, "transport", cfg.Transport, "Transport type: stdio or http")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the MCP server.
func Run(ctx context.Context, cfg Config) error {
	return mcpapp.Run(ctx, cfg.Addr, cfg.HTTPAddr, cfg.Transport)
}
