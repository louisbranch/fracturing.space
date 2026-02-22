package app

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	workerdomain "github.com/louisbranch/fracturing.space/internal/services/worker/domain"
	workerstorage "github.com/louisbranch/fracturing.space/internal/services/worker/storage"
	workersqlite "github.com/louisbranch/fracturing.space/internal/services/worker/storage/sqlite"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// RuntimeConfig controls worker startup, dependencies, and loop behavior.
type RuntimeConfig struct {
	Port              int
	AuthAddr          string
	NotificationsAddr string
	DBPath            string
	Consumer          string
	PollInterval      time.Duration
	LeaseTTL          time.Duration
	MaxAttempts       int
	RetryBackoff      time.Duration
	RetryMaxDelay     time.Duration
	GRPCDialTimeout   time.Duration
}

const (
	defaultWorkerPort = 8089
	defaultWorkerDB   = "data/worker.db"
)

// Run starts worker runtime dependencies and the background processing loop.
func Run(ctx context.Context, cfg RuntimeConfig) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(cfg.AuthAddr) == "" {
		return fmt.Errorf("auth address is required")
	}
	if strings.TrimSpace(cfg.NotificationsAddr) == "" {
		return fmt.Errorf("notifications address is required")
	}
	if cfg.Port <= 0 {
		cfg.Port = defaultWorkerPort
	}
	if strings.TrimSpace(cfg.DBPath) == "" {
		cfg.DBPath = defaultWorkerDB
	}
	if cfg.GRPCDialTimeout <= 0 {
		cfg.GRPCDialTimeout = timeouts.GRPCDial
	}

	if dir := filepath.Dir(cfg.DBPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create worker storage dir: %w", err)
		}
	}

	workerStore, err := workersqlite.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open worker sqlite store: %w", err)
	}
	defer func() {
		if closeErr := workerStore.Close(); closeErr != nil {
			log.Printf("close worker sqlite store: %v", closeErr)
		}
	}()

	authConn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		cfg.AuthAddr,
		cfg.GRPCDialTimeout,
		log.Printf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return fmt.Errorf("dial auth service: %w", err)
	}
	defer func() {
		if closeErr := authConn.Close(); closeErr != nil {
			log.Printf("close auth connection: %v", closeErr)
		}
	}()

	notificationsConn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		cfg.NotificationsAddr,
		cfg.GRPCDialTimeout,
		log.Printf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return fmt.Errorf("dial notifications service: %w", err)
	}
	defer func() {
		if closeErr := notificationsConn.Close(); closeErr != nil {
			log.Printf("close notifications connection: %v", closeErr)
		}
	}()

	authClient := authv1.NewAuthServiceClient(authConn)
	notificationsClient := notificationsv1.NewNotificationServiceClient(notificationsConn)
	handler := workerdomain.NewOnboardingWelcomeHandler(notificationsClient, nil)
	loopConfig := Config{
		Consumer:      cfg.Consumer,
		PollInterval:  cfg.PollInterval,
		LeaseTTL:      cfg.LeaseTTL,
		MaxAttempts:   cfg.MaxAttempts,
		RetryBackoff:  cfg.RetryBackoff,
		RetryMaxDelay: cfg.RetryMaxDelay,
	}
	normalizedLoopConfig := loopConfig.normalized()

	workerLoop := New(
		authClient,
		newAttemptStoreRecorder(workerStore, normalizedLoopConfig.Consumer),
		map[string]EventHandler{
			"auth.signup_completed": handler,
		},
		normalizedLoopConfig,
		nil,
	)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return fmt.Errorf("listen on worker port %d: %w", cfg.Port, err)
	}
	defer listener.Close()

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("worker.runtime", grpc_health_v1.HealthCheckResponse_SERVING)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(listener)
	}()
	defer func() {
		healthServer.Shutdown()
		grpcServer.GracefulStop()
		<-serveErr
	}()

	log.Printf("worker server listening at %v", listener.Addr())
	return workerLoop.Run(ctx)
}

type attemptStoreRecorder struct {
	store    workerstorage.AttemptStore
	consumer string
}

func newAttemptStoreRecorder(store workerstorage.AttemptStore, consumer string) *attemptStoreRecorder {
	normalizedConsumer := strings.TrimSpace(consumer)
	if normalizedConsumer == "" {
		normalizedConsumer = defaultConsumer
	}
	return &attemptStoreRecorder{store: store, consumer: normalizedConsumer}
}

func (r *attemptStoreRecorder) RecordAttempt(ctx context.Context, attempt Attempt) error {
	if r == nil || r.store == nil {
		return nil
	}
	consumer := strings.TrimSpace(r.consumer)
	if consumer == "" {
		consumer = defaultConsumer
	}
	return r.store.RecordAttempt(ctx, workerstorage.AttemptRecord{
		EventID:      attempt.EventID,
		EventType:    attempt.EventType,
		Consumer:     consumer,
		Outcome:      canonicalOutcomeValue(attempt.Outcome),
		AttemptCount: attempt.AttemptCount,
		LastError:    attempt.Error,
		CreatedAt:    attempt.CreatedAt,
	})
}

func canonicalOutcomeValue(outcome authv1.IntegrationOutboxAckOutcome) string {
	switch outcome {
	case authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED:
		return "succeeded"
	case authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY:
		return "retry"
	case authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD:
		return "dead"
	default:
		return "unknown"
	}
}
