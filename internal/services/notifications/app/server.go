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

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/config"
	notificationsservice "github.com/louisbranch/fracturing.space/internal/services/notifications/api/grpc/notifications"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/domain"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage"
	notificationssqlite "github.com/louisbranch/fracturing.space/internal/services/notifications/storage/sqlite"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

const defaultEmailDeliveryWorkerPollInterval = 5 * time.Second

// serverEnv captures env-driven notifications startup settings.
type serverEnv struct {
	DBPath                     string `env:"FRACTURING_SPACE_NOTIFICATIONS_DB_PATH"`
	EmailDeliveryEnabled       string `env:"FRACTURING_SPACE_NOTIFICATIONS_EMAIL_DELIVERY_ENABLED"`
	EmailDeliveryWorkerEnabled string `env:"FRACTURING_SPACE_NOTIFICATIONS_EMAIL_DELIVERY_WORKER_ENABLED"`
	EmailDeliveryWorkerPoll    string `env:"FRACTURING_SPACE_NOTIFICATIONS_EMAIL_DELIVERY_WORKER_POLL_INTERVAL"`
}

func loadServerEnv() serverEnv {
	var cfg serverEnv
	_ = config.ParseEnv(&cfg)
	if strings.TrimSpace(cfg.DBPath) == "" {
		cfg.DBPath = filepath.Join("data", "notifications.db")
	}
	if strings.TrimSpace(cfg.EmailDeliveryWorkerPoll) == "" {
		cfg.EmailDeliveryWorkerPoll = defaultEmailDeliveryWorkerPollInterval.String()
	}
	return cfg
}

func parseBoolEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseDurationEnv(value string, fallback time.Duration) time.Duration {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(trimmed)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

// Server hosts notifications gRPC APIs and optional delivery worker runtime.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server
	store      *notificationssqlite.Store
	closeOnce  sync.Once

	emailDeliveryWorkerEnabled bool
	emailDeliveryWorkerPoll    time.Duration
}

// New creates a configured notifications server listening on the given port.
func New(port int) (*Server, error) {
	return NewWithAddr(fmt.Sprintf(":%d", port))
}

// NewWithAddr creates a configured notifications server listening on the given address.
func NewWithAddr(addr string) (*Server, error) {
	srvEnv := loadServerEnv()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	store, err := openNotificationsStore(srvEnv.DBPath)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}

	emailDeliveryEnabled := parseBoolEnv(srvEnv.EmailDeliveryEnabled)
	adapter := newDomainStoreAdapter(store, store, emailDeliveryEnabled)
	domainService := domain.NewService(adapter, nil, nil)
	apiService := notificationsservice.NewService(domainService)

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	healthServer := health.NewServer()
	notificationsv1.RegisterNotificationServiceServer(grpcServer, apiService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("notifications.v1.NotificationService", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Server{
		listener:                   listener,
		grpcServer:                 grpcServer,
		health:                     healthServer,
		store:                      store,
		emailDeliveryWorkerEnabled: parseBoolEnv(srvEnv.EmailDeliveryWorkerEnabled),
		emailDeliveryWorkerPoll:    parseDurationEnv(srvEnv.EmailDeliveryWorkerPoll, defaultEmailDeliveryWorkerPollInterval),
	}, nil
}

// Addr returns the listener address for the notifications server.
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Run creates and serves a notifications server until context cancellation.
func Run(ctx context.Context, port int) error {
	server, err := New(port)
	if err != nil {
		return err
	}
	return server.Serve(ctx)
}

// RunWithAddr creates and serves a notifications server until context cancellation.
func RunWithAddr(ctx context.Context, addr string) error {
	server, err := NewWithAddr(addr)
	if err != nil {
		return err
	}
	return server.Serve(ctx)
}

// Serve starts notifications gRPC serving and optional background workers.
func (s *Server) Serve(ctx context.Context) error {
	if s == nil {
		return errors.New("server is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	defer s.Close()

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	if s.emailDeliveryWorkerEnabled {
		go s.runEmailDeliveryWorker(workerCtx)
	}

	log.Printf("notifications server listening at %v", s.listener.Addr())
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

// Close releases notifications server resources.
func (s *Server) Close() {
	if s == nil {
		return
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
				log.Printf("close notifications store: %v", err)
			}
		}
	})
}

func openNotificationsStore(path string) (*notificationssqlite.Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}

	store, err := notificationssqlite.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open notifications sqlite store: %w", err)
	}
	return store, nil
}

func (s *Server) runEmailDeliveryWorker(ctx context.Context) {
	pollEvery := s.emailDeliveryWorkerPoll
	if pollEvery <= 0 {
		pollEvery = defaultEmailDeliveryWorkerPollInterval
	}
	ticker := time.NewTicker(pollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.observePendingEmailDeliveries(ctx); err != nil {
				log.Printf("notifications email delivery worker: %v", err)
			}
		}
	}
}

func (s *Server) observePendingEmailDeliveries(ctx context.Context) error {
	if s == nil || s.store == nil {
		return nil
	}
	pending, err := s.store.ListPendingDeliveries(ctx, storage.DeliveryChannelEmail, 50, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("list pending email deliveries: %w", err)
	}
	if len(pending) > 0 {
		log.Printf("notifications email delivery worker observed %d pending deliveries (sender scaffold not configured)", len(pending))
	}
	return nil
}
