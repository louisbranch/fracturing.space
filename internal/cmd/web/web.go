// Package web parses web command flags and boots the browser UI service.
package web

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/discovery"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/web"
)

// Config holds the web command configuration.
type Config struct {
	HTTPAddr            string        `env:"FRACTURING_SPACE_WEB_HTTP_ADDR"           envDefault:"localhost:8080"`
	ChatHTTPAddr        string        `env:"FRACTURING_SPACE_CHAT_HTTP_ADDR"         envDefault:"localhost:8086"`
	AuthBaseURL         string        `env:"FRACTURING_SPACE_WEB_AUTH_BASE_URL"       envDefault:"http://localhost:8084"`
	AuthAddr            string        `env:"FRACTURING_SPACE_WEB_AUTH_ADDR"`
	ConnectionsAddr     string        `env:"FRACTURING_SPACE_CONNECTIONS_ADDR"`
	GameAddr            string        `env:"FRACTURING_SPACE_GAME_ADDR"`
	NotificationsAddr   string        `env:"FRACTURING_SPACE_NOTIFICATIONS_ADDR"`
	AIAddr              string        `env:"FRACTURING_SPACE_AI_ADDR"`
	ListingAddr         string        `env:"FRACTURING_SPACE_LISTING_ADDR"`
	CacheDBPath         string        `env:"FRACTURING_SPACE_WEB_CACHE_DB_PATH"      envDefault:"data/web-cache.db"`
	AssetBaseURL        string        `env:"FRACTURING_SPACE_ASSET_BASE_URL"`
	AssetVersion        string        `env:"FRACTURING_SPACE_ASSET_VERSION"           envDefault:"v1"`
	GRPCDialTimeout     time.Duration `env:"FRACTURING_SPACE_WEB_DIAL_TIMEOUT"        envDefault:"2s"`
	OAuthClientID       string        `env:"FRACTURING_SPACE_WEB_OAUTH_CLIENT_ID"     envDefault:"fracturing-space"`
	CallbackURL         string        `env:"FRACTURING_SPACE_WEB_CALLBACK_URL"`
	AuthTokenURL        string        `env:"FRACTURING_SPACE_WEB_AUTH_TOKEN_URL"`
	Domain              string        `env:"FRACTURING_SPACE_DOMAIN"`
	OAuthResourceSecret string        `env:"FRACTURING_SPACE_WEB_OAUTH_RESOURCE_SECRET"`
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
	cfg.AuthAddr = discovery.OrDefaultGRPCAddr(cfg.AuthAddr, discovery.ServiceAuth)
	cfg.ConnectionsAddr = discovery.OrDefaultGRPCAddr(cfg.ConnectionsAddr, discovery.ServiceConnections)
	cfg.GameAddr = discovery.OrDefaultGRPCAddr(cfg.GameAddr, discovery.ServiceGame)
	cfg.NotificationsAddr = discovery.OrDefaultGRPCAddr(cfg.NotificationsAddr, discovery.ServiceNotifications)
	cfg.ListingAddr = discovery.OrDefaultGRPCAddr(cfg.ListingAddr, discovery.ServiceListing)
	cfg.AIAddr = strings.TrimSpace(cfg.AIAddr)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.ChatHTTPAddr, "chat-http-addr", cfg.ChatHTTPAddr, "Chat HTTP listen address")
	fs.StringVar(&cfg.AuthBaseURL, "auth-base-url", cfg.AuthBaseURL, "Auth service HTTP base URL")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "Auth service gRPC address")
	fs.StringVar(&cfg.ConnectionsAddr, "connections-addr", cfg.ConnectionsAddr, "Connections service gRPC address")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "Game service gRPC address")
	fs.StringVar(&cfg.NotificationsAddr, "notifications-addr", cfg.NotificationsAddr, "Notifications service gRPC address")
	fs.StringVar(&cfg.ListingAddr, "listing-addr", cfg.ListingAddr, "Listing service gRPC address")
	fs.StringVar(&cfg.AIAddr, "ai-addr", cfg.AIAddr, "AI service gRPC address")
	fs.StringVar(&cfg.CacheDBPath, "cache-db-path", cfg.CacheDBPath, "Web cache SQLite path")
	fs.StringVar(&cfg.AssetBaseURL, "asset-base-url", cfg.AssetBaseURL, "Asset base URL for image delivery")
	fs.StringVar(&cfg.AssetVersion, "asset-version", cfg.AssetVersion, "Version prefix for external asset keys")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Run builds and starts the web login surface.
func Run(ctx context.Context, cfg Config) error {
	if err := catalog.ValidateEmbeddedCatalogManifests(); err != nil {
		return err
	}
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceWeb, func(context.Context) error {
		server, err := web.NewServer(web.Config{
			HTTPAddr:             cfg.HTTPAddr,
			ChatHTTPAddr:         cfg.ChatHTTPAddr,
			AuthBaseURL:          cfg.AuthBaseURL,
			AuthAddr:             cfg.AuthAddr,
			ConnectionsAddr:      cfg.ConnectionsAddr,
			GameAddr:             cfg.GameAddr,
			NotificationsAddr:    cfg.NotificationsAddr,
			AIAddr:               cfg.AIAddr,
			ListingAddr:          cfg.ListingAddr,
			CacheDBPath:          cfg.CacheDBPath,
			AssetBaseURL:         cfg.AssetBaseURL,
			AssetManifestVersion: cfg.AssetVersion,
			GRPCDialTimeout:      cfg.GRPCDialTimeout,
			OAuthClientID:        cfg.OAuthClientID,
			CallbackURL:          cfg.CallbackURL,
			AuthTokenURL:         cfg.AuthTokenURL,
			Domain:               cfg.Domain,
			OAuthResourceSecret:  cfg.OAuthResourceSecret,
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
