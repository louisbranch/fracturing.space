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
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
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

// runtimeDependencies keeps the concrete resources opened by cmd/play so they
// can be closed deterministically after the app runtime stops.
type runtimeDependencies struct {
	authMC managedConnResource
	gameMC managedConnResource
	store  transcriptStoreResource
	closed bool
}

func openRuntimeDependencies(ctx context.Context, cfg Config) (runtimeDependencies, playapp.Dependencies, error) {
	return openRuntimeDependenciesWith(ctx, cfg, runtimeDependencyOpeners{
		openManagedConn: func(ctx context.Context, cfg platformgrpc.ManagedConnConfig) (managedConnResource, error) {
			conn, err := platformgrpc.NewManagedConn(ctx, cfg)
			if err != nil {
				return nil, err
			}
			return managedConnAdapter{conn: conn}, nil
		},
		openStore: func(path string) (transcriptStoreResource, error) {
			return playsqlite.Open(path)
		},
	})
}

type managedConnResource interface {
	ClientConn() gogrpc.ClientConnInterface
	Close() error
}

// managedConnAdapter lets the composition root depend on a narrow dial result
// contract so dependency opening can be unit-tested without real gRPC dials.
type managedConnAdapter struct {
	conn *platformgrpc.ManagedConn
}

func (a managedConnAdapter) ClientConn() gogrpc.ClientConnInterface {
	return a.conn.Conn()
}

func (a managedConnAdapter) Close() error {
	return a.conn.Close()
}

// transcriptStoreResource captures the transcript seam plus lifecycle so the
// composition root can wire and clean up the store through one tested contract.
type transcriptStoreResource interface {
	transcript.Store
	Close() error
}

// runtimeDependencyOpeners groups the side-effectful constructors owned by the
// composition root so tests can exercise error handling and cleanup locally.
type runtimeDependencyOpeners struct {
	openManagedConn func(context.Context, platformgrpc.ManagedConnConfig) (managedConnResource, error)
	openStore       func(string) (transcriptStoreResource, error)
}

// openRuntimeDependenciesWith opens the runtime-owned collaborators through
// injectable constructors while preserving the production cleanup semantics.
func openRuntimeDependenciesWith(ctx context.Context, cfg Config, openers runtimeDependencyOpeners) (runtimeDependencies, playapp.Dependencies, error) {
	authMC, err := openers.openManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "auth",
		Addr: cfg.AuthAddr,
		Mode: platformgrpc.ModeRequired,
	})
	if err != nil {
		return runtimeDependencies{}, playapp.Dependencies{}, fmt.Errorf("connect auth: %w", err)
	}
	gameMC, err := openers.openManagedConn(ctx, platformgrpc.ManagedConnConfig{
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
	store, err := openers.openStore(cfg.DBPath)
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
	return resources, dependenciesFromResources(authMC, gameMC, store), nil
}

// dependenciesFromResources builds the app-facing dependency graph after the
// composition root has opened transport/storage resources successfully.
func dependenciesFromResources(authMC managedConnResource, gameMC managedConnResource, store transcriptStoreResource) playapp.Dependencies {
	return playapp.Dependencies{
		Auth:         authv1.NewAuthServiceClient(authMC.ClientConn()),
		Interaction:  gamev1.NewInteractionServiceClient(gameMC.ClientConn()),
		Campaign:     gamev1.NewCampaignServiceClient(gameMC.ClientConn()),
		System:       gamev1.NewSystemServiceClient(gameMC.ClientConn()),
		Participants: gamev1.NewParticipantServiceClient(gameMC.ClientConn()),
		Characters:   gamev1.NewCharacterServiceClient(gameMC.ClientConn()),
		Events:       gamev1.NewEventServiceClient(gameMC.ClientConn()),
		Transcripts:  store,
	}
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
