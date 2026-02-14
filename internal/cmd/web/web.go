package web

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/web"
)

const (
	defaultHTTPAddr    = "localhost:8086"
	defaultAuthBaseURL = "http://localhost:8084"
	defaultAuthAddr    = "localhost:8083"
)

// Config holds the web command configuration.
type Config struct {
	HTTPAddr        string
	AuthBaseURL     string
	AuthAddr        string
	GRPCDialTimeout time.Duration
}

// EnvLookup returns the value for a key when present.
type EnvLookup func(string) (string, bool)

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string, lookup EnvLookup) (Config, error) {
	cfg := Config{
		HTTPAddr:        envOrDefault(lookup, []string{"FRACTURING_SPACE_WEB_HTTP_ADDR"}, defaultHTTPAddr),
		AuthBaseURL:     envOrDefault(lookup, []string{"FRACTURING_SPACE_WEB_AUTH_BASE_URL"}, defaultAuthBaseURL),
		AuthAddr:        envOrDefault(lookup, []string{"FRACTURING_SPACE_WEB_AUTH_ADDR"}, defaultAuthAddr),
		GRPCDialTimeout: 2 * time.Second,
	}

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.AuthBaseURL, "auth-base-url", cfg.AuthBaseURL, "Auth service HTTP base URL")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "Auth service gRPC address")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Run starts the web login server.
func Run(ctx context.Context, cfg Config) error {
	server, err := web.NewServer(web.Config{
		HTTPAddr:        cfg.HTTPAddr,
		AuthBaseURL:     cfg.AuthBaseURL,
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
