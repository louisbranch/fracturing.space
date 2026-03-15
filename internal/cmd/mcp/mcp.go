// Package mcp parses MCP command flags for the internal MCP bridge.
package mcp

import (
	"context"
	"flag"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	mcpapp "github.com/louisbranch/fracturing.space/internal/services/mcp/app"
	mcpservice "github.com/louisbranch/fracturing.space/internal/services/mcp/service"
)

// Config holds MCP command configuration.
type Config struct {
	Addr                string                         `env:"FRACTURING_SPACE_GAME_ADDR"`
	AIAddr              string                         `env:"FRACTURING_SPACE_AI_ADDR"`
	HTTPAddr            string                         `env:"FRACTURING_SPACE_MCP_HTTP_ADDR" envDefault:"localhost:8085"`
	StatusAddr          string                         `env:"FRACTURING_SPACE_STATUS_ADDR"`
	RegistrationProfile mcpservice.RegistrationProfile `env:"FRACTURING_SPACE_MCP_PROFILE"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.Addr = serviceaddr.OrDefaultGRPCAddr(cfg.Addr, serviceaddr.ServiceGame)
	cfg.AIAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AIAddr, serviceaddr.ServiceAI)
	if cfg.StatusAddr != "" {
		cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)
	}

	fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "game server address")
	fs.StringVar(&cfg.AIAddr, "ai-addr", cfg.AIAddr, "ai server address")
	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP server address")
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

		return mcpapp.Run(ctx, cfg.Addr, cfg.AIAddr, cfg.HTTPAddr, cfg.RegistrationProfile)
	})
}
