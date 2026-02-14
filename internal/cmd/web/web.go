package web

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/web"
)

// Config holds the web command configuration.
type Config struct {
	HTTPAddr        string        `env:"FRACTURING_SPACE_WEB_HTTP_ADDR"      envDefault:"localhost:8086"`
	AuthBaseURL     string        `env:"FRACTURING_SPACE_WEB_AUTH_BASE_URL"  envDefault:"http://localhost:8084"`
	AuthAddr        string        `env:"FRACTURING_SPACE_WEB_AUTH_ADDR"      envDefault:"localhost:8083"`
	GRPCDialTimeout time.Duration `env:"FRACTURING_SPACE_WEB_DIAL_TIMEOUT"   envDefault:"2s"`
}

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}
	if cfg.GRPCDialTimeout <= 0 {
		cfg.GRPCDialTimeout = timeouts.GRPCDial
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
