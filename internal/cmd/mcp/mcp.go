package mcp

import (
	"context"
	"flag"
	"strings"

	mcpapp "github.com/louisbranch/fracturing.space/internal/services/mcp/app"
)

// Config holds MCP command configuration.
type Config struct {
	Addr      string
	HTTPAddr  string
	Transport string
}

// EnvLookup returns the value for a key when present.
type EnvLookup func(string) (string, bool)

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string, lookup EnvLookup) (Config, error) {
	cfg := Config{
		Addr:      envOrDefault(lookup, []string{"FRACTURING_SPACE_GAME_ADDR"}, "localhost:8080"),
		HTTPAddr:  envOrDefault(lookup, []string{"FRACTURING_SPACE_MCP_HTTP_ADDR"}, "localhost:8081"),
		Transport: "stdio",
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

func envOrDefault(lookup EnvLookup, keys []string, fallback string) string {
	for _, key := range keys {
		if lookup == nil {
			break
		}
		value, ok := lookup(key)
		if ok {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}
