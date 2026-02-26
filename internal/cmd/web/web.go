// Package web parses command config and boots the web service.
package web

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/discovery"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/web"
)

// Config holds command inputs for web startup.
type Config struct {
	HTTPAddr                  string        `env:"FRACTURING_SPACE_WEB_HTTP_ADDR" envDefault:"localhost:8080"`
	ChatHTTPAddr              string        `env:"FRACTURING_SPACE_CHAT_HTTP_ADDR" envDefault:"localhost:8086"`
	EnableExperimentalModules bool          `env:"FRACTURING_SPACE_WEB_ENABLE_EXPERIMENTAL_MODULES" envDefault:"false"`
	AuthAddr                  string        `env:"FRACTURING_SPACE_AUTH_ADDR"`
	ConnectionsAddr           string        `env:"FRACTURING_SPACE_CONNECTIONS_ADDR"`
	GameAddr                  string        `env:"FRACTURING_SPACE_GAME_ADDR"`
	AIAddr                    string        `env:"FRACTURING_SPACE_AI_ADDR"`
	AssetBaseURL              string        `env:"FRACTURING_SPACE_ASSET_BASE_URL"`
	GRPCDialTimeout           time.Duration `env:"FRACTURING_SPACE_WEB_DIAL_TIMEOUT" envDefault:"2s"`
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
	cfg.AIAddr = discovery.OrDefaultGRPCAddr(cfg.AIAddr, discovery.ServiceAI)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.ChatHTTPAddr, "chat-http-addr", cfg.ChatHTTPAddr, "Chat HTTP listen address")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "Auth service gRPC address")
	fs.StringVar(&cfg.ConnectionsAddr, "connections-addr", cfg.ConnectionsAddr, "Connections service gRPC address")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "Game service gRPC address")
	fs.StringVar(&cfg.AIAddr, "ai-addr", cfg.AIAddr, "AI service gRPC address")
	fs.StringVar(&cfg.AssetBaseURL, "asset-base-url", cfg.AssetBaseURL, "Asset base URL for image delivery")
	fs.BoolVar(&cfg.EnableExperimentalModules, "enable-experimental-modules", cfg.EnableExperimentalModules, "Enable experimental web module surfaces")
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
		authConn, err := platformgrpc.DialWithHealth(ctx, nil, cfg.AuthAddr, cfg.GRPCDialTimeout, log.Printf, platformgrpc.DefaultClientDialOptions()...)
		if err != nil {
			return fmt.Errorf("dial auth gRPC %s: %w", cfg.AuthAddr, err)
		}
		defer authConn.Close()

		connectionsConn, err := platformgrpc.DialWithHealth(ctx, nil, cfg.ConnectionsAddr, cfg.GRPCDialTimeout, log.Printf, platformgrpc.DefaultClientDialOptions()...)
		if err != nil {
			return fmt.Errorf("dial connections gRPC %s: %w", cfg.ConnectionsAddr, err)
		}
		defer connectionsConn.Close()

		gameConn, err := platformgrpc.DialWithHealth(ctx, nil, cfg.GameAddr, cfg.GRPCDialTimeout, log.Printf, platformgrpc.DefaultClientDialOptions()...)
		if err != nil {
			return fmt.Errorf("dial game gRPC %s: %w", cfg.GameAddr, err)
		}
		defer gameConn.Close()

		aiConn, err := platformgrpc.DialWithHealth(ctx, nil, cfg.AIAddr, cfg.GRPCDialTimeout, log.Printf, platformgrpc.DefaultClientDialOptions()...)
		if err != nil {
			return fmt.Errorf("dial ai gRPC %s: %w", cfg.AIAddr, err)
		}
		defer aiConn.Close()

		server, err := web.NewServer(ctx, web.Config{HTTPAddr: cfg.HTTPAddr, AssetBaseURL: cfg.AssetBaseURL, ChatHTTPAddr: cfg.ChatHTTPAddr, EnableExperimentalModules: cfg.EnableExperimentalModules, CampaignClient: statev1.NewCampaignServiceClient(gameConn), ParticipantClient: statev1.NewParticipantServiceClient(gameConn), CharacterClient: statev1.NewCharacterServiceClient(gameConn), SessionClient: statev1.NewSessionServiceClient(gameConn), InviteClient: statev1.NewInviteServiceClient(gameConn), AuthClient: authv1.NewAuthServiceClient(authConn), AccountClient: authv1.NewAccountServiceClient(authConn), ConnectionsClient: connectionsv1.NewConnectionsServiceClient(connectionsConn), CredentialClient: aiv1.NewCredentialServiceClient(aiConn)})
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
