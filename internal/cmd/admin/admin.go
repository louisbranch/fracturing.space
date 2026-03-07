// Package admin parses admin command flags and boots the operator service.
package admin

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/discovery"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/admin"
)

// Config holds the admin command configuration.
type Config struct {
	HTTPAddr            string        `env:"FRACTURING_SPACE_ADMIN_ADDR"                    envDefault:":8081"`
	GRPCAddr            string        `env:"FRACTURING_SPACE_GAME_ADDR"`
	AuthAddr            string        `env:"FRACTURING_SPACE_AUTH_ADDR"`
	GRPCDialTimeout     time.Duration `env:"FRACTURING_SPACE_ADMIN_DIAL_TIMEOUT"             envDefault:"2s"`
	StatusAddr          string        `env:"FRACTURING_SPACE_STATUS_ADDR"`
	AuthIntrospectURL   string        `env:"FRACTURING_SPACE_ADMIN_AUTH_INTROSPECT_URL"`
	OAuthResourceSecret string        `env:"FRACTURING_SPACE_ADMIN_OAUTH_RESOURCE_SECRET"`
	LoginURL            string        `env:"FRACTURING_SPACE_ADMIN_LOGIN_URL"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.GRPCDialTimeout <= 0 {
		cfg.GRPCDialTimeout = timeouts.GRPCDial
	}
	cfg.GRPCAddr = discovery.OrDefaultGRPCAddr(cfg.GRPCAddr, discovery.ServiceGame)
	cfg.AuthAddr = discovery.OrDefaultGRPCAddr(cfg.AuthAddr, discovery.ServiceAuth)
	cfg.StatusAddr = discovery.OrDefaultGRPCAddr(cfg.StatusAddr, discovery.ServiceStatus)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.GRPCAddr, "grpc-addr", cfg.GRPCAddr, "game server address")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "auth server address")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
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

	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceAdmin, func(context.Context) error {
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
		reporter := platformstatus.NewReporter("admin", statusClient)
		reporter.Register("admin.dashboard", platformstatus.Operational)
		reporter.Register("admin.game.integration", platformstatus.Operational)
		reporter.Register("admin.auth.integration", platformstatus.Operational)
		stopReporter := reporter.Start(ctx)
		defer stopReporter()

		server, err := admin.NewServer(ctx, admin.Config{
			HTTPAddr:        cfg.HTTPAddr,
			GRPCAddr:        cfg.GRPCAddr,
			AuthAddr:        cfg.AuthAddr,
			StatusAddr:      cfg.StatusAddr,
			GRPCDialTimeout: cfg.GRPCDialTimeout,
			AuthConfig:      authCfg,
		})
		if err != nil {
			return fmt.Errorf("init admin server: %w", err)
		}
		defer server.Close()

		if err := server.ListenAndServe(ctx); err != nil {
			return fmt.Errorf("serve admin: %w", err)
		}
		return nil
	})
}
