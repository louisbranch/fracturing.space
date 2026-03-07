// Package web parses command config and boots the web service.
package web

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/web"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// Config holds command inputs for web startup.
type Config struct {
	HTTPAddr            string        `env:"FRACTURING_SPACE_WEB_HTTP_ADDR" envDefault:"localhost:8080"`
	ChatHTTPAddr        string        `env:"FRACTURING_SPACE_CHAT_HTTP_ADDR" envDefault:"localhost:8086"`
	TrustForwardedProto bool          `env:"FRACTURING_SPACE_WEB_TRUST_FORWARDED_PROTO" envDefault:"false"`
	AuthAddr            string        `env:"FRACTURING_SPACE_AUTH_ADDR"`
	SocialAddr          string        `env:"FRACTURING_SPACE_SOCIAL_ADDR"`
	GameAddr            string        `env:"FRACTURING_SPACE_GAME_ADDR"`
	AIAddr              string        `env:"FRACTURING_SPACE_AI_ADDR"`
	DiscoveryAddr       string        `env:"FRACTURING_SPACE_DISCOVERY_ADDR"`
	NotificationsAddr   string        `env:"FRACTURING_SPACE_NOTIFICATIONS_ADDR"`
	UserHubAddr         string        `env:"FRACTURING_SPACE_USERHUB_ADDR"`
	StatusAddr          string        `env:"FRACTURING_SPACE_STATUS_ADDR"`
	AssetBaseURL        string        `env:"FRACTURING_SPACE_ASSET_BASE_URL"`
	GRPCDialTimeout     time.Duration `env:"FRACTURING_SPACE_WEB_DIAL_TIMEOUT" envDefault:"2s"`
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
	cfg.AuthAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AuthAddr, serviceaddr.ServiceAuth)
	cfg.SocialAddr = serviceaddr.OrDefaultGRPCAddr(cfg.SocialAddr, serviceaddr.ServiceSocial)
	cfg.GameAddr = serviceaddr.OrDefaultGRPCAddr(cfg.GameAddr, serviceaddr.ServiceGame)
	cfg.AIAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AIAddr, serviceaddr.ServiceAI)
	cfg.DiscoveryAddr = serviceaddr.OrDefaultGRPCAddr(cfg.DiscoveryAddr, serviceaddr.ServiceDiscovery)
	cfg.NotificationsAddr = serviceaddr.OrDefaultGRPCAddr(cfg.NotificationsAddr, serviceaddr.ServiceNotifications)
	cfg.UserHubAddr = serviceaddr.OrDefaultGRPCAddr(cfg.UserHubAddr, serviceaddr.ServiceUserHub)
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.ChatHTTPAddr, "chat-http-addr", cfg.ChatHTTPAddr, "Chat HTTP listen address")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "Auth service gRPC address")
	fs.StringVar(&cfg.SocialAddr, "social-addr", cfg.SocialAddr, "Social service gRPC address")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "Game service gRPC address")
	fs.StringVar(&cfg.AIAddr, "ai-addr", cfg.AIAddr, "AI service gRPC address")
	fs.StringVar(&cfg.DiscoveryAddr, "discovery-addr", cfg.DiscoveryAddr, "Discovery service gRPC address")
	fs.StringVar(&cfg.NotificationsAddr, "notifications-addr", cfg.NotificationsAddr, "Notifications service gRPC address")
	fs.StringVar(&cfg.UserHubAddr, "userhub-addr", cfg.UserHubAddr, "Userhub service gRPC address")
	fs.StringVar(&cfg.AssetBaseURL, "asset-base-url", cfg.AssetBaseURL, "Asset base URL for image delivery")
	fs.BoolVar(&cfg.TrustForwardedProto, "trust-forwarded-proto", cfg.TrustForwardedProto, "Trust X-Forwarded-Proto when resolving request scheme")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run starts the web service process.
func Run(ctx context.Context, cfg Config) error {
	if err := catalog.ValidateEmbeddedCatalogManifests(); err != nil {
		return err
	}
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceWeb, func(context.Context) error {
		requirements := dependencyRequirements(cfg)
		dependencies, dependencyConns, statuses, bootstrapErr := bootstrapDependencies(
			ctx,
			requirements,
			cfg.AssetBaseURL,
			cfg.GRPCDialTimeout,
			dialDependency,
		)
		for _, statusLine := range formatDependencyStatusLines(statuses) {
			log.Printf("web startup: %s", statusLine)
		}
		for _, warning := range dependencyStatusWarnings(requirements, statuses) {
			log.Printf("web startup: %s", warning)
		}
		if bootstrapErr != nil {
			closeDependencyConnections(dependencyConns)
			return fmt.Errorf("init web dependency graph: %w", bootstrapErr)
		}
		defer closeDependencyConnections(dependencyConns)

		statusClient, stopStatusReporter := startStatusReporter(ctx, cfg.StatusAddr, cfg.GRPCDialTimeout, requirements, statuses)
		defer stopStatusReporter()

		// Share the status client with modules for dashboard health queries.
		dependencies.Modules.StatusClient = statusClient

		server, err := web.NewServer(ctx, web.Config{
			HTTPAddr:            cfg.HTTPAddr,
			ChatHTTPAddr:        cfg.ChatHTTPAddr,
			RequestSchemePolicy: requestmeta.SchemePolicy{TrustForwardedProto: cfg.TrustForwardedProto},
			Dependencies:        &dependencies,
		})
		if err != nil {
			return fmt.Errorf("init web server: %w", err)
		}
		defer server.Close()
		if err := server.ListenAndServe(ctx); err != nil {
			return fmt.Errorf("serve web: %w", err)
		}
		return nil
	})
}
