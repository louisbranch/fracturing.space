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
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/discovery"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/web"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	grpc "google.golang.org/grpc"
)

// Config holds command inputs for web startup.
type Config struct {
	HTTPAddr                  string        `env:"FRACTURING_SPACE_WEB_HTTP_ADDR" envDefault:"localhost:8080"`
	ChatHTTPAddr              string        `env:"FRACTURING_SPACE_CHAT_HTTP_ADDR" envDefault:"localhost:8086"`
	TrustForwardedProto       bool          `env:"FRACTURING_SPACE_WEB_TRUST_FORWARDED_PROTO" envDefault:"false"`
	EnableExperimentalModules bool          `env:"FRACTURING_SPACE_WEB_ENABLE_EXPERIMENTAL_MODULES" envDefault:"false"`
	AuthAddr                  string        `env:"FRACTURING_SPACE_AUTH_ADDR"`
	SocialAddr                string        `env:"FRACTURING_SPACE_SOCIAL_ADDR"`
	GameAddr                  string        `env:"FRACTURING_SPACE_GAME_ADDR"`
	AIAddr                    string        `env:"FRACTURING_SPACE_AI_ADDR"`
	NotificationsAddr         string        `env:"FRACTURING_SPACE_NOTIFICATIONS_ADDR"`
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
	cfg.NotificationsAddr = discovery.OrDefaultGRPCAddr(cfg.NotificationsAddr, discovery.ServiceNotifications)
	cfg.UserHubAddr = discovery.OrDefaultGRPCAddr(cfg.UserHubAddr, discovery.ServiceUserHub)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	fs.StringVar(&cfg.ChatHTTPAddr, "chat-http-addr", cfg.ChatHTTPAddr, "Chat HTTP listen address")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "Auth service gRPC address")
	fs.StringVar(&cfg.SocialAddr, "social-addr", cfg.SocialAddr, "Social service gRPC address")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "Game service gRPC address")
	fs.StringVar(&cfg.AIAddr, "ai-addr", cfg.AIAddr, "AI service gRPC address")
	fs.StringVar(&cfg.NotificationsAddr, "notifications-addr", cfg.NotificationsAddr, "Notifications service gRPC address")
	fs.StringVar(&cfg.UserHubAddr, "userhub-addr", cfg.UserHubAddr, "Userhub service gRPC address")
	fs.StringVar(&cfg.AssetBaseURL, "asset-base-url", cfg.AssetBaseURL, "Asset base URL for image delivery")
	fs.BoolVar(&cfg.TrustForwardedProto, "trust-forwarded-proto", cfg.TrustForwardedProto, "Trust X-Forwarded-Proto when resolving request scheme")
	fs.BoolVar(&cfg.EnableExperimentalModules, "enable-experimental-modules", cfg.EnableExperimentalModules, "Enable experimental web module surfaces")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

type grpcDialer func(context.Context, string, time.Duration) (*grpc.ClientConn, error)

type dependencyRequirement struct {
	name     string
	address  string
	setInput func(*web.PrincipalDependencies, *modules.Dependencies, *grpc.ClientConn)
}

func bootstrapDependencies(
	ctx context.Context,
	cfg Config,
	dialer grpcDialer,
) (web.DependencyBundle, []*grpc.ClientConn, []string, error) {
	var principal web.PrincipalDependencies
	principal.AssetBaseURL = cfg.AssetBaseURL
	var modDeps modules.Dependencies
	modDeps.AssetBaseURL = cfg.AssetBaseURL
	conns := []*grpc.ClientConn{}
	warnings := []string{}

	deps := []dependencyRequirement{
		{
			name:    "auth",
			address: cfg.AuthAddr,
			setInput: func(p *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				authClient := authv1.NewAuthServiceClient(conn)
				accountClient := authv1.NewAccountServiceClient(conn)
				p.SessionClient = authClient
				p.AccountClient = accountClient
				m.AuthClient = authClient
				m.AccountClient = accountClient
			},
		},
		{
			name:    "social",
			address: cfg.SocialAddr,
			setInput: func(p *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				socialClient := socialv1.NewSocialServiceClient(conn)
				p.SocialClient = socialClient
				m.SocialClient = socialClient
			},
		},
		{
			name:    "game",
			address: cfg.GameAddr,
			setInput: func(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				m.CampaignClient = statev1.NewCampaignServiceClient(conn)
				m.ParticipantClient = statev1.NewParticipantServiceClient(conn)
				m.CharacterClient = statev1.NewCharacterServiceClient(conn)
				m.DaggerheartContentClient = daggerheartv1.NewDaggerheartContentServiceClient(conn)
				m.SessionClient = statev1.NewSessionServiceClient(conn)
				m.InviteClient = statev1.NewInviteServiceClient(conn)
				m.AuthorizationClient = statev1.NewAuthorizationServiceClient(conn)
			},
		},
		{
			name:    "ai",
			address: cfg.AIAddr,
			setInput: func(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				m.CredentialClient = aiv1.NewCredentialServiceClient(conn)
			},
		},
		{
			name:    "userhub",
			address: cfg.UserHubAddr,
			setInput: func(_ *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				m.UserHubClient = userhubv1.NewUserHubServiceClient(conn)
			},
		},
		{
			name:    "notifications",
			address: cfg.NotificationsAddr,
			setInput: func(p *web.PrincipalDependencies, m *modules.Dependencies, conn *grpc.ClientConn) {
				notificationClient := notificationsv1.NewNotificationServiceClient(conn)
				p.NotificationClient = notificationClient
				m.NotificationClient = notificationClient
			},
		},
	}

	for _, dep := range deps {
		conn, err := dialer(ctx, dep.address, cfg.GRPCDialTimeout)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s dependency at %s unavailable: %v", dep.name, dep.address, err))
			continue
		}
		if conn == nil {
			warnings = append(warnings, fmt.Sprintf("%s dependency at %s unavailable", dep.name, dep.address))
			continue
		}
		dep.setInput(&principal, &modDeps, conn)
		conns = append(conns, conn)
	}

	bundle := web.DependencyBundle{Principal: principal, Modules: modDeps}
	return bundle, conns, warnings, nil
}

func closeDependencyConnections(conns []*grpc.ClientConn) {
	for _, conn := range conns {
		if conn == nil {
			continue
		}
		_ = conn.Close()
	}
}

func dialDependency(
	ctx context.Context,
	address string,
	timeout time.Duration,
) (*grpc.ClientConn, error) {
	return platformgrpc.DialWithHealth(ctx, nil, address, timeout, log.Printf, platformgrpc.DefaultClientDialOptions()...)
}

// Run starts the web service process.
func Run(ctx context.Context, cfg Config) error {
	if err := catalog.ValidateEmbeddedCatalogManifests(); err != nil {
		return err
	}
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceWeb, func(context.Context) error {
		dependencies, dependencyConns, warnings, err := bootstrapDependencies(ctx, cfg, dialDependency)
		if err != nil {
			return fmt.Errorf("init web dependency graph: %w", err)
		}
		defer closeDependencyConnections(dependencyConns)
		for _, warning := range warnings {
			log.Printf("web startup: %s", warning)
		}

		server, err := web.NewServer(ctx, web.Config{
			HTTPAddr:                  cfg.HTTPAddr,
			ChatHTTPAddr:              cfg.ChatHTTPAddr,
			EnableExperimentalModules: cfg.EnableExperimentalModules,
			RequestSchemePolicy:       requestmeta.SchemePolicy{TrustForwardedProto: cfg.TrustForwardedProto},
			Dependencies:              &dependencies,
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
