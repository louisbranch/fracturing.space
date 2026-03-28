package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/config"
	authservice "github.com/louisbranch/fracturing.space/internal/services/auth/api/grpc/auth"
	"github.com/louisbranch/fracturing.space/internal/services/auth/oauth"
	authsqlite "github.com/louisbranch/fracturing.space/internal/services/auth/storage/sqlite"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// authServerEnv captures env-driven auth startup settings.
type authServerEnv struct {
	DBPath string `env:"FRACTURING_SPACE_AUTH_DB_PATH"`
}

type runtimeDeps struct {
	loadEnv         func() authServerEnv
	loadOAuthConfig func() oauth.Config
	listen          func(network, address string) (net.Listener, error)
	openStore       func(string) (*authsqlite.Store, error)
	newOAuthServer  func(oauth.Config, *oauth.Store, oauth.UserStore) *oauth.Server
	logf            func(format string, args ...any)
}

var defaultRuntimeDeps = runtimeDeps{
	loadEnv:         loadAuthServerEnv,
	loadOAuthConfig: oauth.LoadConfigFromEnv,
	listen:          net.Listen,
	openStore:       openAuthStore,
	newOAuthServer:  oauth.NewServer,
	logf:            log.Printf,
}

func loadAuthServerEnv() authServerEnv {
	var cfg authServerEnv
	_ = config.ParseEnv(&cfg)
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join("data", "auth.db")
	}
	return cfg
}

// Server hosts the auth process and keeps gRPC plus OAuth HTTP flows aligned
// around one identity store.
type Server struct {
	listener     net.Listener
	grpcServer   *grpc.Server
	health       *health.Server
	store        *authsqlite.Store
	httpListener net.Listener
	httpServer   *http.Server
	oauthStore   *oauth.Store
	oauthServer  *oauth.Server
	closeOnce    sync.Once
	logf         func(format string, args ...any)
}

// New creates a configured auth server and binds identity transport boundaries.
//
// It initializes one SQLite store first so both gRPC handlers and OAuth routes
// read and write from the same identity ledger.
func New(port int, httpAddr string) (*Server, error) {
	return newWithDeps(port, httpAddr, defaultRuntimeDeps)
}

func newWithDeps(port int, httpAddr string, deps runtimeDeps) (*Server, error) {
	if deps.loadEnv == nil {
		return nil, errors.New("auth server env loader is required")
	}
	if deps.loadOAuthConfig == nil {
		return nil, errors.New("auth oauth config loader is required")
	}
	if deps.listen == nil {
		return nil, errors.New("auth listener constructor is required")
	}
	if deps.openStore == nil {
		return nil, errors.New("auth store opener is required")
	}
	if deps.newOAuthServer == nil {
		return nil, errors.New("auth oauth server constructor is required")
	}
	if deps.logf == nil {
		deps.logf = log.Printf
	}

	srvEnv := deps.loadEnv()
	listener, err := deps.listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("Listen on port %d: %w", port, err)
	}
	store, err := deps.openStore(srvEnv.DBPath)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}

	oauthStore := oauth.NewStore(store.DB())
	oauthConfig := deps.loadOAuthConfig()
	if oauthConfig.Issuer == "" {
		oauthConfig.Issuer = defaultOAuthIssuer(httpAddr)
	}
	var httpListener net.Listener
	var httpServer *http.Server
	var oauthServer *oauth.Server
	if strings.TrimSpace(httpAddr) != "" {
		httpListener, err = deps.listen("tcp", httpAddr)
		if err != nil {
			_ = listener.Close()
			_ = store.Close()
			return nil, fmt.Errorf("Listen on HTTP address %s: %w", httpAddr, err)
		}
		mux := http.NewServeMux()
		oauthServer = deps.newOAuthServer(oauthConfig, oauthStore, store)
		if err := oauthServer.RegisterRoutes(mux); err != nil {
			_ = httpListener.Close()
			_ = listener.Close()
			_ = store.Close()
			return nil, fmt.Errorf("Register OAuth routes: %w", err)
		}
		httpServer = &http.Server{Handler: mux}
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	authService := authservice.NewAuthService(store, store, oauthStore)
	statisticsService := authservice.NewStatisticsService(store)
	accountService := authservice.NewAccountService(store)
	healthServer := health.NewServer()
	authv1.RegisterAuthServiceServer(grpcServer, authService)
	authv1.RegisterStatisticsServiceServer(grpcServer, statisticsService)
	authv1.RegisterAccountServiceServer(grpcServer, accountService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("auth.v1.AuthService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("auth.v1.StatisticsService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("auth.v1.AccountService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Server{
		listener:     listener,
		grpcServer:   grpcServer,
		health:       healthServer,
		store:        store,
		httpListener: httpListener,
		httpServer:   httpServer,
		oauthStore:   oauthStore,
		oauthServer:  oauthServer,
		logf:         deps.logf,
	}, nil
}

// Addr returns the gRPC listener address for the auth server.
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Run creates and serves an auth server until the context ends.
func Run(ctx context.Context, port int, httpAddr string) error {
	if ctx == nil {
		return errors.New("Context is required.")
	}
	grpcServer, err := New(port, httpAddr)
	if err != nil {
		return err
	}
	return grpcServer.Serve(ctx)
}

// Serve starts both transport surfaces and blocks until shutdown.
//
// This is the lifecycle boundary where authentication state, token cleanup, and
// transport shutdown are coordinated together.
func (s *Server) Serve(ctx context.Context) error {
	if s == nil {
		return errors.New("Server is nil.")
	}
	if ctx == nil {
		return errors.New("Context is required.")
	}
	serverCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer s.Close()

	if s.oauthServer != nil {
		s.oauthServer.StartCleanup(serverCtx, 5*time.Minute)
	}

	logf := s.logf
	if logf == nil {
		logf = log.Printf
	}
	logf("auth server listening at %v", s.listener.Addr())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.grpcServer.Serve(s.listener)
	}()

	httpErr := make(chan error, 1)
	if s.httpServer != nil && s.httpListener != nil {
		logf("auth HTTP server listening at %v", s.httpListener.Addr())
		go func() {
			httpErr <- s.httpServer.Serve(s.httpListener)
		}()
	}

	handleErr := func(err error) error {
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("Serve gRPC: %w", err)
	}

	shutdownGRPC := func() {
		if s.health != nil {
			s.health.Shutdown()
		}
		s.grpcServer.GracefulStop()
	}
	shutdownHTTP := func() {
		if s.httpServer != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = s.httpServer.Shutdown(shutdownCtx)
		}
	}

	select {
	case <-ctx.Done():
		shutdownGRPC()
		shutdownHTTP()
		err := <-serveErr
		return handleErr(err)
	case err := <-serveErr:
		shutdownHTTP()
		return handleErr(err)
	case err := <-httpErr:
		if err == http.ErrServerClosed {
			return nil
		}
		shutdownGRPC()
		grpcErr := <-serveErr
		if handled := handleErr(grpcErr); handled != nil {
			return handled
		}
		return fmt.Errorf("Serve HTTP: %w", err)
	}
}

// openAuthStore opens the auth SQLite database and prepares any missing path.
func openAuthStore(path string) (*authsqlite.Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("Create storage directory: %w", err)
		}
	}

	store, err := authsqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Open auth SQLite store: %w", err)
	}
	return store, nil
}

// Close releases transport and storage resources for the auth runtime.
func (s *Server) Close() {
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
			if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				logf("close auth listener: %v", err)
			}
		}
		if s.httpServer != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = s.httpServer.Shutdown(shutdownCtx)
		}
		if s.httpListener != nil {
			if err := s.httpListener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				logf("close auth http listener: %v", err)
			}
		}
		if s.store != nil {
			if err := s.store.Close(); err != nil {
				logf("close auth store: %v", err)
			}
		}
	})
}

// defaultOAuthIssuer infers the OAuth issuer URL for metadata when none is set.
func defaultOAuthIssuer(httpAddr string) string {
	addr := strings.TrimSpace(httpAddr)
	if addr == "" {
		return ""
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	return "http://" + addr
}
