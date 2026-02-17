// Package admin parses admin command flags and boots the operator service.
package admin

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/admin"
)

// Config holds the admin command configuration.
type Config struct {
	HTTPAddr            string        `env:"FRACTURING_SPACE_ADMIN_ADDR"                    envDefault:":8082"`
	GRPCAddr            string        `env:"FRACTURING_SPACE_GAME_ADDR"                     envDefault:"localhost:8080"`
	AuthAddr            string        `env:"FRACTURING_SPACE_AUTH_ADDR"                     envDefault:"localhost:8083"`
	GRPCDialTimeout     time.Duration `env:"FRACTURING_SPACE_ADMIN_DIAL_TIMEOUT"             envDefault:"2s"`
	AuthIntrospectURL   string        `env:"FRACTURING_SPACE_ADMIN_AUTH_INTROSPECT_URL"`
	OAuthResourceSecret string        `env:"FRACTURING_SPACE_ADMIN_OAUTH_RESOURCE_SECRET"`
	LoginURL            string        `env:"FRACTURING_SPACE_ADMIN_LOGIN_URL"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := config.ParseEnv(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.GRPCDialTimeout <= 0 {
		cfg.GRPCDialTimeout = timeouts.GRPCDial
	}

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.GRPCAddr, "grpc-addr", cfg.GRPCAddr, "game server address")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "auth server address")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Run creates the admin control-plane and starts it for the current process.
func Run(ctx context.Context, cfg Config) error {
	var authCfg *admin.AuthConfig
	if strings.TrimSpace(cfg.AuthIntrospectURL) != "" && strings.TrimSpace(cfg.LoginURL) != "" {
		authCfg = &admin.AuthConfig{
			IntrospectURL:  cfg.AuthIntrospectURL,
			ResourceSecret: cfg.OAuthResourceSecret,
			LoginURL:       cfg.LoginURL,
		}
	}

	server, err := admin.NewServer(ctx, admin.Config{
		HTTPAddr:        cfg.HTTPAddr,
		GRPCAddr:        cfg.GRPCAddr,
		AuthAddr:        cfg.AuthAddr,
		GRPCDialTimeout: cfg.GRPCDialTimeout,
		AuthConfig:      authCfg,
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
