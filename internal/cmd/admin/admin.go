package admin

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/admin"
)

const (
	defaultHTTPAddr = ":8082"
	defaultGRPCAddr = "localhost:8080"
	defaultAuthAddr = "localhost:8083"
)

// Config holds the admin command configuration.
type Config struct {
	HTTPAddr        string
	GRPCAddr        string
	AuthAddr        string
	GRPCDialTimeout time.Duration
}

// EnvLookup returns the value for a key when present.
type EnvLookup func(string) (string, bool)

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string, lookup EnvLookup) (Config, error) {
	cfg := Config{
		HTTPAddr:        envOrDefault(lookup, []string{"FRACTURING_SPACE_ADMIN_ADDR"}, defaultHTTPAddr),
		GRPCAddr:        envOrDefault(lookup, []string{"FRACTURING_SPACE_GAME_ADDR"}, defaultGRPCAddr),
		AuthAddr:        envOrDefault(lookup, []string{"FRACTURING_SPACE_AUTH_ADDR"}, defaultAuthAddr),
		GRPCDialTimeout: 2 * time.Second,
	}

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.GRPCAddr, "grpc-addr", cfg.GRPCAddr, "game server address")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "auth server address")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Run starts the admin server.
func Run(ctx context.Context, cfg Config) error {
	server, err := admin.NewServer(ctx, admin.Config{
		HTTPAddr:        cfg.HTTPAddr,
		GRPCAddr:        cfg.GRPCAddr,
		AuthAddr:        cfg.AuthAddr,
		GRPCDialTimeout: cfg.GRPCDialTimeout,
	})
	if err != nil {
		return fmt.Errorf("init web server: %w", err)
	}
	defer server.Close()

	if err := server.ListenAndServe(ctx); err != nil {
		return fmt.Errorf("serve web: %w", err)
	}
	return nil
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
