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

	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/config"
	discoveryservice "github.com/louisbranch/fracturing.space/internal/services/discovery/api/grpc/discovery"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
	discoverysqlite "github.com/louisbranch/fracturing.space/internal/services/discovery/storage/sqlite"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type serverEnv struct {
	DBPath string `env:"FRACTURING_SPACE_DISCOVERY_DB_PATH"`
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
}

// New creates a configured discovery server listening on the provided port.
func New(port int) (*Server, error) {
	return NewWithAddr(fmt.Sprintf(":%d", port))
}

// NewWithAddr creates a configured discovery server for the provided address.
func NewWithAddr(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	env := loadServerEnv()
	store, err := openDiscoveryStore(env.DBPath)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	if err := bootstrapBuiltinCatalog(store); err != nil {
		_ = listener.Close()
		_ = store.Close()
		return nil, fmt.Errorf("bootstrap builtin catalog: %w", err)
	}

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	apiService := discoveryservice.NewService(store)
	healthServer := health.NewServer()
	discoveryv1.RegisterDiscoveryServiceServer(grpcServer, apiService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("discovery.v1.DiscoveryService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Server{
		listener:   listener,
		grpcServer: grpcServer,
		health:     healthServer,
		store:      store,
	}, nil
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

	log.Printf("discovery server listening at %v", s.listener.Addr())
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

// Close releases discovery server resources.
func (s *Server) Close() {
	if s == nil {
		return
	}
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
			log.Printf("close discovery store: %v", err)
		}
	}
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
		if err := store.CreateDiscoveryEntry(ctx, entry); err != nil {
			if errors.Is(err, storage.ErrAlreadyExists) {
				continue
			}
			return fmt.Errorf("create builtin entry %q: %w", entry.EntryID, err)
		}
		log.Printf("bootstrapped builtin entry: %s", entry.EntryID)
	}
	return nil
}
