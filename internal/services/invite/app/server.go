// Package app wires the invite runtime and gRPC lifecycle.
package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/config"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	inviteservice "github.com/louisbranch/fracturing.space/internal/services/invite/api/grpc/invite"
	invitesqlite "github.com/louisbranch/fracturing.space/internal/services/invite/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// Config holds invite service runtime configuration.
type Config struct {
	Port     int
	GameAddr string
	AuthAddr string
}

type runtimeDeps struct {
	loadEnv   func() serverEnv
	listen    func(network, address string) (net.Listener, error)
	openStore func(string) (*invitesqlite.Store, error)
	dialGame  func(context.Context, string, func(string, ...any)) (*grpc.ClientConn, error)
	dialAuth  func(context.Context, string, func(string, ...any)) (*grpc.ClientConn, error)
	logf      func(format string, args ...any)
}

var defaultRuntimeDeps = runtimeDeps{
	loadEnv:   loadServerEnv,
	listen:    net.Listen,
	openStore: openInviteStore,
	dialGame: func(ctx context.Context, addr string, logf func(string, ...any)) (*grpc.ClientConn, error) {
		gameDialOpts := append(
			platformgrpc.LenientDialOptions(),
			grpc.WithChainUnaryInterceptor(grpcauthctx.ServiceIDUnaryClientInterceptor(serviceaddr.ServiceInvite)),
		)
		return platformgrpc.DialLenient(ctx, addr, logf, gameDialOpts...), nil
	},
	dialAuth: func(ctx context.Context, addr string, logf func(string, ...any)) (*grpc.ClientConn, error) {
		return platformgrpc.DialLenient(ctx, addr, logf), nil
	},
	logf: log.Printf,
}

type serverEnv struct {
	DBPath string `env:"FRACTURING_SPACE_INVITE_DB_PATH"`
}

func loadServerEnv() serverEnv {
	var cfg serverEnv
	_ = config.ParseEnv(&cfg)
	if strings.TrimSpace(cfg.DBPath) == "" {
		cfg.DBPath = filepath.Join("data", "invite.db")
	}
	return cfg
}

// Server hosts the invite gRPC API and storage lifecycle.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server
	store      *invitesqlite.Store
	gameConn   *grpc.ClientConn
	authConn   *grpc.ClientConn
	closeOnce  sync.Once
	logf       func(format string, args ...any)
}

// Run creates and serves an invite server until context cancellation.
func Run(ctx context.Context, cfg Config) error {
	server, err := NewWithAddr(ctx, fmt.Sprintf(":%d", cfg.Port), cfg.GameAddr, cfg.AuthAddr)
	if err != nil {
		return err
	}
	return server.Serve(ctx)
}

// NewWithAddr creates a ready-to-serve invite server on the given address.
// Use "127.0.0.1:0" for tests to let the OS pick an ephemeral port; call
// Addr() after construction to discover the assigned address.
func NewWithAddr(ctx context.Context, addr, gameAddr, authAddr string) (*Server, error) {
	return newWithDeps(ctx, addr, gameAddr, authAddr, defaultRuntimeDeps)
}

func newWithDeps(ctx context.Context, addr, gameAddr, authAddr string, deps runtimeDeps) (*Server, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if deps.loadEnv == nil {
		return nil, errors.New("invite server env loader is required")
	}
	if deps.listen == nil {
		return nil, errors.New("invite listener constructor is required")
	}
	if deps.openStore == nil {
		return nil, errors.New("invite store opener is required")
	}
	if deps.dialGame == nil {
		return nil, errors.New("invite game dialer is required")
	}
	if deps.dialAuth == nil {
		return nil, errors.New("invite auth dialer is required")
	}
	if deps.logf == nil {
		deps.logf = log.Printf
	}

	listener, err := deps.listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	srvEnv := deps.loadEnv()
	store, err := deps.openStore(srvEnv.DBPath)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}

	logf := func(format string, args ...any) {
		deps.logf("[invite] "+format, args...)
	}
	gameConn, err := deps.dialGame(ctx, gameAddr, logf)
	if err != nil {
		_ = listener.Close()
		_ = store.Close()
		return nil, fmt.Errorf("dial game service: %w", err)
	}
	authConn, err := deps.dialAuth(ctx, authAddr, logf)
	if err != nil {
		_ = listener.Close()
		_ = store.Close()
		if gameConn != nil {
			_ = gameConn.Close()
		}
		return nil, fmt.Errorf("dial auth service: %w", err)
	}

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))

	svc := inviteservice.NewService(inviteservice.Deps{
		Store:        store,
		Outbox:       store,
		Game:         gamev1.NewParticipantServiceClient(gameConn),
		GameCampaign: gamev1.NewCampaignServiceClient(gameConn),
		Auth:         authv1.NewAuthServiceClient(authConn),
		IDGenerator:  id.NewID,
		Verifier:     joingrant.EnvVerifier{},
	})

	healthServer := health.NewServer()
	invitev1.RegisterInviteServiceServer(grpcServer, svc)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("invite.v1.InviteService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Server{
		listener:   listener,
		grpcServer: grpcServer,
		health:     healthServer,
		store:      store,
		gameConn:   gameConn,
		authConn:   authConn,
		logf:       deps.logf,
	}, nil
}

// Addr returns the address the server is listening on.
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Serve starts the gRPC server and blocks until ctx is cancelled or an error occurs.
func (s *Server) Serve(ctx context.Context) error {
	defer s.close()

	logf := s.logf
	if logf == nil {
		logf = log.Printf
	}
	logf("invite server listening at %v", s.listener.Addr())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.grpcServer.Serve(s.listener)
	}()

	select {
	case <-ctx.Done():
		if s.health != nil {
			s.health.Shutdown()
		}
		s.grpcServer.GracefulStop()
		err := <-serveErr
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve gRPC: %w", err)
	case err := <-serveErr:
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve gRPC: %w", err)
	}
}

func (s *Server) close() {
	if s == nil {
		return
	}
	logf := s.logf
	if logf == nil {
		logf = log.Printf
	}
	s.closeOnce.Do(func() {
		if s.health != nil {
			s.health.Shutdown()
		}
		if s.grpcServer != nil {
			s.grpcServer.Stop()
		}
		if s.listener != nil {
			_ = s.listener.Close()
		}
		if s.gameConn != nil {
			_ = s.gameConn.Close()
		}
		if s.authConn != nil {
			_ = s.authConn.Close()
		}
		if s.store != nil {
			if err := s.store.Close(); err != nil {
				logf("close invite store: %v", err)
			}
		}
	})
}

func openInviteStore(path string) (*invitesqlite.Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}
	store, err := invitesqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open invite sqlite store: %w", err)
	}
	return store, nil
}
