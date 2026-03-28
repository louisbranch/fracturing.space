// Package server wires the discovery runtime and gRPC lifecycle.
package server

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
	"time"

	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/config"
	discoveryservice "github.com/louisbranch/fracturing.space/internal/services/discovery/api/grpc/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/catalog"
	discoverysqlite "github.com/louisbranch/fracturing.space/internal/services/discovery/storage/sqlite"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type runtimeDeps struct {
	loadEnv          func() serverEnv
	listen           func(network, address string) (net.Listener, error)
	openStore        func(string) (*discoverysqlite.Store, error)
	bootstrapCatalog func(*discoverysqlite.Store) error
	openGameConn     func(string) (*grpc.ClientConn, error)
	buildReconciler  func(*discoverysqlite.Store, *grpc.ClientConn) func(context.Context)
	logf             func(format string, args ...any)
}

var defaultRuntimeDeps = runtimeDeps{
	loadEnv:          loadServerEnv,
	listen:           net.Listen,
	openStore:        openDiscoveryStore,
	bootstrapCatalog: bootstrapBuiltinCatalog,
	openGameConn:     openGameConn,
	buildReconciler: func(store *discoverysqlite.Store, gameConn *grpc.ClientConn) func(context.Context) {
		if gameConn == nil {
			return nil
		}
		return starterReconciler(
			store,
			gamev1.NewCampaignServiceClient(gameConn),
			gamev1.NewCharacterServiceClient(gameConn),
		)
	},
	logf: log.Printf,
}

type serverEnv struct {
	DBPath   string `env:"FRACTURING_SPACE_DISCOVERY_DB_PATH"`
	GameAddr string `env:"FRACTURING_SPACE_GAME_ADDR"`
}

func loadServerEnv() serverEnv {
	var cfg serverEnv
	_ = config.ParseEnv(&cfg)
	if strings.TrimSpace(cfg.DBPath) == "" {
		cfg.DBPath = filepath.Join("data", "discovery.db")
	}
	return cfg
}

// Server hosts the discovery gRPC API and storage lifecycle.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server
	store      *discoverysqlite.Store
	gameConn   *grpc.ClientConn
	reconcile  func(context.Context)
	closeOnce  sync.Once
	logf       func(format string, args ...any)
}

// New creates a configured discovery server listening on the provided port.
func New(port int) (*Server, error) {
	return NewWithAddr(fmt.Sprintf(":%d", port))
}

// NewWithAddr creates a configured discovery server for the provided address.
func NewWithAddr(addr string) (*Server, error) {
	return newWithDeps(addr, defaultRuntimeDeps)
}

func newWithDeps(addr string, deps runtimeDeps) (*Server, error) {
	if deps.loadEnv == nil {
		return nil, errors.New("discovery server env loader is required")
	}
	if deps.listen == nil {
		return nil, errors.New("discovery listener constructor is required")
	}
	if deps.openStore == nil {
		return nil, errors.New("discovery store opener is required")
	}
	if deps.bootstrapCatalog == nil {
		return nil, errors.New("discovery catalog bootstrapper is required")
	}
	if deps.openGameConn == nil {
		return nil, errors.New("discovery game dialer is required")
	}
	if deps.buildReconciler == nil {
		return nil, errors.New("discovery reconciler builder is required")
	}
	if deps.logf == nil {
		deps.logf = log.Printf
	}

	listener, err := deps.listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	env := deps.loadEnv()
	store, err := deps.openStore(env.DBPath)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	if err := deps.bootstrapCatalog(store); err != nil {
		_ = listener.Close()
		_ = store.Close()
		return nil, fmt.Errorf("bootstrap builtin catalog: %w", err)
	}
	gameConn, err := deps.openGameConn(env.GameAddr)
	if err != nil {
		_ = listener.Close()
		_ = store.Close()
		return nil, fmt.Errorf("open game connection: %w", err)
	}
	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	apiService := discoveryservice.NewService(store)
	healthServer := health.NewServer()
	discoveryv1.RegisterDiscoveryServiceServer(grpcServer, apiService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("discovery.v1.DiscoveryService", grpc_health_v1.HealthCheckResponse_SERVING)

	server := &Server{
		listener:   listener,
		grpcServer: grpcServer,
		health:     healthServer,
		store:      store,
		gameConn:   gameConn,
		logf:       deps.logf,
	}
	server.reconcile = deps.buildReconciler(store, gameConn)
	if gameConn == nil {
		deps.logf("discovery starter reconciliation skipped: FRACTURING_SPACE_GAME_ADDR is unset")
	}

	return server, nil
}

// Addr returns the listener address for the server.
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Run creates and serves a discovery server until context cancellation.
func Run(ctx context.Context, port int) error {
	server, err := New(port)
	if err != nil {
		return err
	}
	return server.Serve(ctx)
}

// Serve starts the gRPC server until context cancellation.
func (s *Server) Serve(ctx context.Context) error {
	if s == nil {
		return errors.New("server is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	defer s.Close()

	logf := s.logf
	if logf == nil {
		logf = log.Printf
	}
	logf("discovery server listening at %v", s.listener.Addr())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.grpcServer.Serve(s.listener)
	}()
	if s.reconcile != nil {
		go s.reconcile(ctx)
	}

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

func starterReconciler(
	store *discoverysqlite.Store,
	campaignClient gamev1.CampaignServiceClient,
	characterClient gamev1.CharacterServiceClient,
) func(context.Context) {
	return func(ctx context.Context) {
		const retryDelay = 2 * time.Second

		for {
			if ctx != nil && ctx.Err() != nil {
				return
			}
			err := reconcileBuiltinStarterTemplates(ctx, store, campaignClient, characterClient)
			if err == nil {
				log.Printf("discovery starter reconciliation completed")
				return
			}
			if !isRetryableStarterReconciliationError(err) {
				log.Printf("discovery starter reconciliation stopped: %v", err)
				return
			}
			log.Printf("discovery starter reconciliation pending catalog readiness: %v", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
			}
		}
	}
}

// Close releases discovery server resources.
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
			_ = s.listener.Close()
		}
		if s.store != nil {
			if err := s.store.Close(); err != nil {
				logf("close discovery store: %v", err)
			}
		}
		if s.gameConn != nil {
			if err := s.gameConn.Close(); err != nil {
				logf("close discovery game conn: %v", err)
			}
		}
	})
}

func openDiscoveryStore(path string) (*discoverysqlite.Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}
	store, err := discoverysqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open discovery sqlite store: %w", err)
	}
	return store, nil
}

// bootstrapBuiltinCatalog inserts embedded discovery entries into the store if
// they do not already exist. Already-existing entries are silently skipped.
func bootstrapBuiltinCatalog(store *discoverysqlite.Store) error {
	entries, err := catalog.BuiltinEntries()
	if err != nil {
		return fmt.Errorf("load builtin entries: %w", err)
	}
	ctx := context.Background()
	for _, entry := range entries {
		if err := store.UpsertBuiltinDiscoveryEntry(ctx, entry); err != nil {
			return fmt.Errorf("upsert builtin entry %q: %w", entry.EntryID, err)
		}
		log.Printf("bootstrapped builtin entry: %s", entry.EntryID)
	}
	return nil
}

func openGameConn(addr string) (*grpc.ClientConn, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, nil
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return conn, nil
}
