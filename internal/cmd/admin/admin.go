// Package admin parses admin command flags and boots the operator service.
package admin

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/services/admin"
)

// Config holds the admin command configuration.
type Config struct {
	HTTPAddr            string `env:"FRACTURING_SPACE_ADMIN_ADDR"                    envDefault:":8081"`
	GRPCAddr            string `env:"FRACTURING_SPACE_GAME_ADDR"`
	AuthAddr            string `env:"FRACTURING_SPACE_AUTH_ADDR"`
	InviteAddr          string `env:"FRACTURING_SPACE_ADMIN_INVITE_ADDR"`
	StatusAddr          string `env:"FRACTURING_SPACE_STATUS_ADDR"`
	AuthIntrospectURL   string `env:"FRACTURING_SPACE_ADMIN_AUTH_INTROSPECT_URL"`
	OAuthResourceSecret string `env:"FRACTURING_SPACE_ADMIN_OAUTH_RESOURCE_SECRET"`
	LoginURL            string `env:"FRACTURING_SPACE_ADMIN_LOGIN_URL"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.GRPCAddr = serviceaddr.OrDefaultGRPCAddr(cfg.GRPCAddr, serviceaddr.ServiceGame)
	cfg.AuthAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AuthAddr, serviceaddr.ServiceAuth)
	cfg.InviteAddr = serviceaddr.OrDefaultGRPCAddr(cfg.InviteAddr, serviceaddr.ServiceInvite)
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.GRPCAddr, "grpc-addr", cfg.GRPCAddr, "game server address")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "auth server address")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// logConfiguredAddresses logs the dependency addresses for startup diagnostics.
func logConfiguredAddresses(cfg Config) {
	if addr := strings.TrimSpace(cfg.GRPCAddr); addr != "" {
		log.Printf("admin startup: dependency=game address=%s", addr)
	}
	if addr := strings.TrimSpace(cfg.AuthAddr); addr != "" {
		log.Printf("admin startup: dependency=auth address=%s", addr)
	}
	if addr := strings.TrimSpace(cfg.InviteAddr); addr != "" {
		log.Printf("admin startup: dependency=invite address=%s", addr)
	}
	if addr := strings.TrimSpace(cfg.StatusAddr); addr != "" {
		log.Printf("admin startup: dependency=status address=%s", addr)
	}
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

	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceAdmin, func(context.Context) error {
		reporter := platformstatus.NewReporter("admin", nil)
		reporter.Register("admin.dashboard", platformstatus.Operational)

		logConfiguredAddresses(cfg)

		server, err := admin.NewServer(ctx, admin.Config{
			HTTPAddr:       cfg.HTTPAddr,
			GRPCAddr:       cfg.GRPCAddr,
			AuthAddr:       cfg.AuthAddr,
			InviteAddr:     cfg.InviteAddr,
			StatusAddr:     cfg.StatusAddr,
			AuthConfig:     authCfg,
			StatusReporter: reporter,
		})
		if err != nil {
			return fmt.Errorf("init admin server: %w", err)
		}
		defer server.Close()

		stopReporter := reporter.Start(ctx)
		defer stopReporter()

		if err := server.ListenAndServe(ctx); err != nil {
			return fmt.Errorf("serve admin: %w", err)
		}
		return nil
	})
}
