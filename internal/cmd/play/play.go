// Package play parses play command flags and composes transport entrypoints.
package play

import (
	"context"
	"errors"
	"flag"
	"fmt"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	playapp "github.com/louisbranch/fracturing.space/internal/services/play/app"
	playsqlite "github.com/louisbranch/fracturing.space/internal/services/play/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	gogrpc "google.golang.org/grpc"
)

// Config holds play command configuration.
type Config struct {
	HTTPAddr            string `env:"FRACTURING_SPACE_PLAY_HTTP_ADDR" envDefault:":8094"`
	WebHTTPAddr         string `env:"FRACTURING_SPACE_WEB_HTTP_ADDR"`
	AuthAddr            string `env:"FRACTURING_SPACE_AUTH_ADDR"`
	GameAddr            string `env:"FRACTURING_SPACE_GAME_ADDR"`
	StatusAddr          string `env:"FRACTURING_SPACE_STATUS_ADDR"`
	DBPath              string `env:"FRACTURING_SPACE_PLAY_DB_PATH" envDefault:"data/play.db"`
	PlayUIDevServerURL  string `env:"FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL"`
	TrustForwardedProto bool   `env:"FRACTURING_SPACE_PLAY_TRUST_FORWARDED_PROTO" envDefault:"false"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}
	cfg.WebHTTPAddr = serviceaddr.OrDefaultHTTPAddr(cfg.WebHTTPAddr, serviceaddr.ServiceWeb)
	cfg.AuthAddr = serviceaddr.OrDefaultGRPCAddr(cfg.AuthAddr, serviceaddr.ServiceAuth)
	cfg.GameAddr = serviceaddr.OrDefaultGRPCAddr(cfg.GameAddr, serviceaddr.ServiceGame)
	cfg.StatusAddr = serviceaddr.OrDefaultGRPCAddr(cfg.StatusAddr, serviceaddr.ServiceStatus)

	fs.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "play HTTP listen address")
	fs.StringVar(&cfg.WebHTTPAddr, "web-http-addr", cfg.WebHTTPAddr, "web HTTP listen address for browser fallback links")
	fs.StringVar(&cfg.AuthAddr, "auth-addr", cfg.AuthAddr, "auth service gRPC address")
	fs.StringVar(&cfg.GameAddr, "game-addr", cfg.GameAddr, "game service gRPC address")
	fs.StringVar(&cfg.DBPath, "db-path", cfg.DBPath, "play SQLite database path")
	fs.StringVar(&cfg.PlayUIDevServerURL, "ui-dev-server-url", cfg.PlayUIDevServerURL, "optional play UI dev server URL")
	fs.BoolVar(&cfg.TrustForwardedProto, "trust-forwarded-proto", cfg.TrustForwardedProto, "trust X-Forwarded-Proto when resolving request scheme")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run builds the play app and starts browser-facing runtime behavior.
func Run(ctx context.Context, cfg Config) error {
	launchGrantCfg, err := playlaunchgrant.LoadConfigFromEnv(nil)
	if err != nil {
		return fmt.Errorf("load play launch grant config: %w", err)
	}
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServicePlay, func(context.Context) error {
		stopReporter := entrypoint.StartStatusReporter(
			ctx,
			"play",
			cfg.StatusAddr,
			entrypoint.Capability("play.http", platformstatus.Operational),
			entrypoint.Capability("play.game.integration", platformstatus.Operational),
			entrypoint.Capability("play.auth.integration", platformstatus.Operational),
		)
		defer stopReporter()

		resources, deps, err := openRuntimeDependencies(ctx, cfg)
		if err != nil {
			return fmt.Errorf("open play dependencies: %w", err)
		}
		defer func() { _ = resources.Close() }()

		server, err := playapp.NewServer(playapp.Config{
			HTTPAddr:            cfg.HTTPAddr,
			WebHTTPAddr:         cfg.WebHTTPAddr,
			PlayUIDevServerURL:  cfg.PlayUIDevServerURL,
			RequestSchemePolicy: requestmeta.SchemePolicy{TrustForwardedProto: cfg.TrustForwardedProto},
			LaunchGrant:         launchGrantCfg,
		}, deps)
		if err != nil {
			return fmt.Errorf("init play server: %w", err)
		}
		defer server.Close()
		if err := server.ListenAndServe(ctx); err != nil {
			return fmt.Errorf("serve play: %w", err)
		}
		return nil
	})
}

type runtimeDependencies struct {
	authMC *platformgrpc.ManagedConn
	gameMC *platformgrpc.ManagedConn
	store  *playsqlite.Store
	closed bool
}

func openRuntimeDependencies(ctx context.Context, cfg Config) (runtimeDependencies, playapp.Dependencies, error) {
	authMC, err := platformgrpc.NewManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "auth",
		Addr: cfg.AuthAddr,
		Mode: platformgrpc.ModeRequired,
	})
	if err != nil {
		return runtimeDependencies{}, playapp.Dependencies{}, fmt.Errorf("connect auth: %w", err)
	}
	gameMC, err := platformgrpc.NewManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "game",
		Addr: cfg.GameAddr,
		Mode: platformgrpc.ModeRequired,
		DialOpts: append(
			platformgrpc.LenientDialOptions(),
			gogrpc.WithChainUnaryInterceptor(grpcauthctx.ServiceIDUnaryClientInterceptor(serviceaddr.ServicePlay)),
			gogrpc.WithChainStreamInterceptor(grpcauthctx.ServiceIDStreamClientInterceptor(serviceaddr.ServicePlay)),
		),
	})
	if err != nil {
		_ = authMC.Close()
		return runtimeDependencies{}, playapp.Dependencies{}, fmt.Errorf("connect game: %w", err)
	}
	store, err := playsqlite.Open(cfg.DBPath)
	if err != nil {
		_ = gameMC.Close()
		_ = authMC.Close()
		return runtimeDependencies{}, playapp.Dependencies{}, fmt.Errorf("open play transcript store: %w", err)
	}
	resources := runtimeDependencies{
		authMC: authMC,
		gameMC: gameMC,
		store:  store,
	}
	return resources, playapp.Dependencies{
		Auth:        authv1.NewAuthServiceClient(authMC.Conn()),
		Interaction: gamev1.NewInteractionServiceClient(gameMC.Conn()),
		Campaign:    gamev1.NewCampaignServiceClient(gameMC.Conn()),
		System:      gamev1.NewSystemServiceClient(gameMC.Conn()),
		Events:      gamev1.NewEventServiceClient(gameMC.Conn()),
		Transcripts: store,
	}, nil
}

func (r *runtimeDependencies) Close() error {
	if r == nil || r.closed {
		return nil
	}
	r.closed = true
	return errors.Join(
		closeIfPresent(r.store),
		closeIfPresent(r.gameMC),
		closeIfPresent(r.authMC),
	)
}

type closer interface {
	Close() error
}

func closeIfPresent(value closer) error {
	if value == nil {
		return nil
	}
	return value.Close()
}
