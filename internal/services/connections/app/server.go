// Package server wires the connections runtime and gRPC lifecycle.
package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/config"
	connectionsservice "github.com/louisbranch/fracturing.space/internal/services/connections/api/grpc/connections"
	connectionsstorage "github.com/louisbranch/fracturing.space/internal/services/connections/storage"
	connectionssqlite "github.com/louisbranch/fracturing.space/internal/services/connections/storage/sqlite"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type serverEnv struct {
	DBPath          string `env:"FRACTURING_SPACE_CONNECTIONS_DB_PATH"`
	AuthDBPath      string `env:"FRACTURING_SPACE_AUTH_DB_PATH"`
	MigrateAuthData string `env:"FRACTURING_SPACE_CONNECTIONS_MIGRATE_AUTH_CONTACTS"`
}

func loadServerEnv() serverEnv {
	var cfg serverEnv
	_ = config.ParseEnv(&cfg)
	if strings.TrimSpace(cfg.DBPath) == "" {
		cfg.DBPath = filepath.Join("data", "connections.db")
	}
	if strings.TrimSpace(cfg.AuthDBPath) == "" {
		cfg.AuthDBPath = filepath.Join("data", "auth.db")
	}
	if strings.TrimSpace(cfg.MigrateAuthData) == "" {
		cfg.MigrateAuthData = "true"
	}
	return cfg
}

// Server hosts the connections gRPC API and storage lifecycle.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server
	store      *connectionssqlite.Store
}

// New creates a configured connections server listening on the provided port.
func New(port int) (*Server, error) {
	return NewWithAddr(fmt.Sprintf(":%d", port))
}

// NewWithAddr creates a configured connections server for the provided address.
func NewWithAddr(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}
	srvEnv := loadServerEnv()
	store, err := openConnectionsStore(srvEnv.DBPath)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	if parseBoolEnv(srvEnv.MigrateAuthData) {
		if err := migrateContactsFromAuth(context.Background(), srvEnv.AuthDBPath, store); err != nil {
			_ = listener.Close()
			_ = store.Close()
			return nil, fmt.Errorf("migrate contacts from auth: %w", err)
		}
	}

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	apiService := connectionsservice.NewService(store)
	healthServer := health.NewServer()
	connectionsv1.RegisterConnectionsServiceServer(grpcServer, apiService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("connections.v1.ConnectionsService", grpc_health_v1.HealthCheckResponse_SERVING)

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

// Run creates and serves a connections server until context cancellation.
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

	log.Printf("connections server listening at %v", s.listener.Addr())
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

// Close releases connections server resources.
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
			log.Printf("close connections store: %v", err)
		}
	}
}

func openConnectionsStore(path string) (*connectionssqlite.Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}
	store, err := connectionssqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open connections sqlite store: %w", err)
	}
	return store, nil
}

func parseBoolEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func migrateContactsFromAuth(ctx context.Context, authDBPath string, destination connectionsstorage.ContactStore) error {
	if strings.TrimSpace(authDBPath) == "" || destination == nil {
		return nil
	}
	if _, err := os.Stat(authDBPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat auth db: %w", err)
	}

	authDB, err := sql.Open("sqlite", filepath.Clean(authDBPath)+"?_foreign_keys=ON&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("open auth db: %w", err)
	}
	defer func() { _ = authDB.Close() }()

	if err := authDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping auth db: %w", err)
	}

	// `user_contacts` may be absent in fresh auth schemas after clean-slate cutover.
	rows, err := authDB.QueryContext(
		ctx,
		`SELECT owner_user_id, contact_user_id, created_at, updated_at
		 FROM user_contacts`,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return nil
		}
		return fmt.Errorf("query auth user_contacts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			ownerUserID   string
			contactUserID string
			createdAtMS   int64
			updatedAtMS   int64
		)
		if err := rows.Scan(&ownerUserID, &contactUserID, &createdAtMS, &updatedAtMS); err != nil {
			return fmt.Errorf("scan auth user_contacts row: %w", err)
		}
		contact := connectionsstorage.Contact{
			OwnerUserID:   strings.TrimSpace(ownerUserID),
			ContactUserID: strings.TrimSpace(contactUserID),
			CreatedAt:     time.UnixMilli(createdAtMS).UTC(),
			UpdatedAt:     time.UnixMilli(updatedAtMS).UTC(),
		}
		if err := destination.PutContact(ctx, contact); err != nil {
			return fmt.Errorf("persist migrated contact %s->%s: %w", contact.OwnerUserID, contact.ContactUserID, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate auth user_contacts rows: %w", err)
	}
	return nil
}
