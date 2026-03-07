// Package mcp parses MCP command flags and selects stdio or HTTP transport.
package mcp

import (
	"context"
	"flag"
	"log"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/discovery"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
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
	cfg.Addr = discovery.OrDefaultGRPCAddr(cfg.Addr, discovery.ServiceGame)
	cfg.StatusAddr = discovery.OrDefaultGRPCAddr(cfg.StatusAddr, discovery.ServiceStatus)

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
		// Status reporter.
		statusConn := platformgrpc.DialLenient(ctx, cfg.StatusAddr, log.Printf)
		if statusConn != nil {
			defer func() {
				if err := statusConn.Close(); err != nil {
					log.Printf("close status connection: %v", err)
				}
			}()
		}
		var statusClient statusv1.StatusServiceClient
		if statusConn != nil {
			statusClient = statusv1.NewStatusServiceClient(statusConn)
		}
		reporter := platformstatus.NewReporter("mcp", statusClient)
		reporter.Register("mcp.tools", platformstatus.Operational)
		reporter.Register("mcp.game.integration", platformstatus.Operational)
		stopReporter := reporter.Start(ctx)
		defer stopReporter()

		return mcpapp.Run(ctx, cfg.Addr, cfg.HTTPAddr, cfg.Transport)
	})
}
