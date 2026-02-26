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
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
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
	SocialAddr                string        `env:"FRACTURING_SPACE_SOCIAL_ADDR"`
	GameAddr                  string        `env:"FRACTURING_SPACE_GAME_ADDR"`
	AIAddr                    string        `env:"FRACTURING_SPACE_AI_ADDR"`
	UserHubAddr               string        `env:"FRACTURING_SPACE_USERHUB_ADDR"`
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
	cfg.SocialAddr = discovery.OrDefaultGRPCAddr(cfg.SocialAddr, discovery.ServiceSocial)
	cfg.GameAddr = discovery.OrDefaultGRPCAddr(cfg.GameAddr, discovery.ServiceGame)
	cfg.AIAddr = discovery.OrDefaultGRPCAddr(cfg.AIAddr, discovery.ServiceAI)
	cfg.UserHubAddr = discovery.OrDefaultGRPCAddr(cfg.UserHubAddr, discovery.ServiceUserHub)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.ChatHTTPAddr, "chat-http-addr", cfg.ChatHTTPAddr, "Chat HTTP listen address")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "Auth service gRPC address")
	fs.StringVar(&cfg.SocialAddr, "social-addr", cfg.SocialAddr, "Social service gRPC address")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "Game service gRPC address")
	fs.StringVar(&cfg.AIAddr, "ai-addr", cfg.AIAddr, "AI service gRPC address")
	fs.StringVar(&cfg.UserHubAddr, "userhub-addr", cfg.UserHubAddr, "Userhub service gRPC address")
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

		socialConn, err := platformgrpc.DialWithHealth(ctx, nil, cfg.SocialAddr, cfg.GRPCDialTimeout, log.Printf, platformgrpc.DefaultClientDialOptions()...)
		if err != nil {
			return fmt.Errorf("dial social gRPC %s: %w", cfg.SocialAddr, err)
		}
		defer socialConn.Close()

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

		userHubConn, err := platformgrpc.DialWithHealth(ctx, nil, cfg.UserHubAddr, cfg.GRPCDialTimeout, log.Printf, platformgrpc.DefaultClientDialOptions()...)
		if err != nil {
			return fmt.Errorf("dial userhub gRPC %s: %w", cfg.UserHubAddr, err)
		}
		defer userHubConn.Close()

		server, err := web.NewServer(ctx, web.Config{HTTPAddr: cfg.HTTPAddr, AssetBaseURL: cfg.AssetBaseURL, ChatHTTPAddr: cfg.ChatHTTPAddr, EnableExperimentalModules: cfg.EnableExperimentalModules, CampaignClient: statev1.NewCampaignServiceClient(gameConn), ParticipantClient: statev1.NewParticipantServiceClient(gameConn), CharacterClient: statev1.NewCharacterServiceClient(gameConn), SessionClient: statev1.NewSessionServiceClient(gameConn), InviteClient: statev1.NewInviteServiceClient(gameConn), AuthorizationClient: statev1.NewAuthorizationServiceClient(gameConn), AuthClient: authv1.NewAuthServiceClient(authConn), AccountClient: authv1.NewAccountServiceClient(authConn), SocialClient: socialv1.NewSocialServiceClient(socialConn), CredentialClient: aiv1.NewCredentialServiceClient(aiConn), UserHubClient: userhubv1.NewUserHubServiceClient(userHubConn)})
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
