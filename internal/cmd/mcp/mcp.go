// Package mcp parses MCP command flags and selects stdio or HTTP transport.
package mcp

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/platform/otel"
	mcpapp "github.com/louisbranch/fracturing.space/internal/services/mcp/app"
)

// Config holds MCP command configuration.
type Config struct {
	Addr      string `env:"FRACTURING_SPACE_GAME_ADDR"     envDefault:"localhost:8080"`
	HTTPAddr  string `env:"FRACTURING_SPACE_MCP_HTTP_ADDR" envDefault:"localhost:8081"`
	Transport string `env:"FRACTURING_SPACE_MCP_TRANSPORT" envDefault:"stdio"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := config.ParseEnv(&cfg); err != nil {
		return Config{}, err
	}

	fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "game server address")
	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP server address (for HTTP transport)")
	fs.StringVar(&cfg.Transport, "transport", cfg.Transport, "Transport type: stdio or http")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the MCP protocol adapter.
func Run(ctx context.Context, cfg Config) error {
	shutdown, err := otel.Setup(ctx, "mcp")
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

	return mcpapp.Run(ctx, cfg.Addr, cfg.HTTPAddr, cfg.Transport)
}
