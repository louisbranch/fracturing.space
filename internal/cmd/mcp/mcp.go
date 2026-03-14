// Package mcp parses MCP command flags and selects stdio or HTTP transport.
package mcp

import (
	"context"
	"flag"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	mcpapp "github.com/louisbranch/fracturing.space/internal/services/mcp/app"
)

// Config holds MCP command configuration.
type Config struct {
	Addr       string `env:"FRACTURING_SPACE_GAME_ADDR"`
	HTTPAddr   string `env:"FRACTURING_SPACE_MCP_HTTP_ADDR" envDefault:"localhost:8085"`
	Transport  string `env:"FRACTURING_SPACE_MCP_TRANSPORT" envDefault:"stdio"`
	StatusAddr string `env:"FRACTURING_SPACE_STATUS_ADDR"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.Addr = serviceaddr.OrDefaultGRPCAddr(cfg.Addr, serviceaddr.ServiceGame)
	if cfg.StatusAddr != "" {
		cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)
	}

	fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "game server address")
	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP server address (for HTTP transport)")
	fs.StringVar(&cfg.Transport, "transport", cfg.Transport, "Transport type: stdio or http")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the MCP protocol adapter.
func Run(ctx context.Context, cfg Config) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceMCP, func(context.Context) error {
		stopReporter := entrypoint.StartStatusReporter(
			ctx,
			"mcp",
			cfg.StatusAddr,
			entrypoint.Capability("mcp.tools", platformstatus.Operational),
			entrypoint.Capability("mcp.game.integration", platformstatus.Operational),
		)
		defer stopReporter()

		return mcpapp.Run(ctx, cfg.Addr, cfg.HTTPAddr, cfg.Transport)
	})
}
