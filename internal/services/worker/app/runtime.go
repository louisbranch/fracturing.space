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
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
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
	SocialAddr        string
	NotificationsAddr string
	DBPath            string
	Consumer          string
	PollInterval      time.Duration
	LeaseTTL          time.Duration
	MaxAttempts       int
	RetryBackoff      time.Duration
	RetryMaxDelay     time.Duration
}

const (
	defaultWorkerPort = 8089
	defaultWorkerDB   = "data/worker.db"
)

var (
	newManagedConn  = platformgrpc.NewManagedConn
	openSQLiteStore = workersqlite.Open
	listenTCP       = net.Listen
)

type workerLoop interface {
	Run(ctx context.Context) error
}

// Runtime wires worker transport, dependency clients, and loop execution.
type Runtime struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server

	loop workerLoop

	store           *workersqlite.Store
	authMc          *platformgrpc.ManagedConn
	notificationsMc *platformgrpc.ManagedConn
	socialMc        *platformgrpc.ManagedConn

	closeOnce sync.Once
}

// Run starts worker runtime dependencies and the background processing loop.
func Run(ctx context.Context, cfg RuntimeConfig) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	runtime, err := NewRuntime(ctx, cfg)
	if err != nil {
		return err
	}
	return runtime.Serve(ctx)
}

// NewRuntime constructs a configured worker runtime with dependency wiring.
func NewRuntime(ctx context.Context, cfg RuntimeConfig) (*Runtime, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	normalized, err := normalizeRuntimeConfig(cfg)
	if err != nil {
		return nil, err
	}

	if dir := filepath.Dir(normalized.DBPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create worker storage dir: %w", err)
		}
	}

	workerStore, err := openSQLiteStore(normalized.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open worker sqlite store: %w", err)
	}

	logf := func(format string, args ...any) {
		log.Printf(format, args...)
	}

	authMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "auth",
		Addr: normalized.AuthAddr,
		Mode: platformgrpc.ModeRequired,
		Logf: logf,
	})
	if err != nil {
		if closeErr := workerStore.Close(); closeErr != nil {
			log.Printf("close worker sqlite store: %v", closeErr)
		}
		return nil, fmt.Errorf("worker: managed conn auth: %w", err)
	}

	notificationsMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "notifications",
		Addr: normalized.NotificationsAddr,
		Mode: platformgrpc.ModeRequired,
		Logf: logf,
	})
	if err != nil {
		closeManagedConn(authMc, "auth")
		if closeErr := workerStore.Close(); closeErr != nil {
			log.Printf("close worker sqlite store: %v", closeErr)
		}
		return nil, fmt.Errorf("worker: managed conn notifications: %w", err)
	}

	socialMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "social",
		Addr: normalized.SocialAddr,
		Mode: platformgrpc.ModeRequired,
		Logf: logf,
	})
	if err != nil {
		closeManagedConn(notificationsMc, "notifications")
		closeManagedConn(authMc, "auth")
		if closeErr := workerStore.Close(); closeErr != nil {
			log.Printf("close worker sqlite store: %v", closeErr)
		}
		return nil, fmt.Errorf("worker: managed conn social: %w", err)
	}

	authClient := authv1.NewAuthServiceClient(authMc.Conn())
	notificationsClient := notificationsv1.NewNotificationServiceClient(notificationsMc.Conn())
	socialClient := socialv1.NewSocialServiceClient(socialMc.Conn())
	profileHandler := workerdomain.NewSignupSocialProfileHandler(socialClient)
	welcomeHandler := workerdomain.NewOnboardingWelcomeHandler(notificationsClient, nil)
	handler := fanoutEventHandlers(profileHandler, welcomeHandler)
	loopConfig := Config{
		Consumer:      normalized.Consumer,
		PollInterval:  normalized.PollInterval,
		LeaseTTL:      normalized.LeaseTTL,
		MaxAttempts:   normalized.MaxAttempts,
		RetryBackoff:  normalized.RetryBackoff,
		RetryMaxDelay: normalized.RetryMaxDelay,
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

	listener, err := listenTCP("tcp", fmt.Sprintf(":%d", normalized.Port))
	if err != nil {
		closeManagedConn(socialMc, "social")
		closeManagedConn(notificationsMc, "notifications")
		closeManagedConn(authMc, "auth")
		if closeErr := workerStore.Close(); closeErr != nil {
			log.Printf("close worker sqlite store: %v", closeErr)
		}
		return nil, fmt.Errorf("listen on worker port %d: %w", normalized.Port, err)
	}

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("worker.runtime", grpc_health_v1.HealthCheckResponse_SERVING)

	return &Runtime{
		listener:        listener,
		grpcServer:      grpcServer,
		health:          healthServer,
		loop:            workerLoop,
		store:           workerStore,
		authMc:          authMc,
		notificationsMc: notificationsMc,
		socialMc:        socialMc,
	}, nil
}

// Addr returns the bound worker runtime listener address.
func (r *Runtime) Addr() string {
	if r == nil || r.listener == nil {
		return ""
	}
	return r.listener.Addr().String()
}

// Serve runs health transport and worker loop until cancellation or failure.
func (r *Runtime) Serve(ctx context.Context) error {
	if r == nil {
		return errors.New("runtime is nil")
	}
	if r.listener == nil || r.grpcServer == nil || r.loop == nil {
		return errors.New("runtime is not configured")
	}
	if ctx == nil {
		return errors.New("context is required")
	}
	defer r.Close()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- r.grpcServer.Serve(r.listener)
	}()

	loopErr := make(chan error, 1)
	go func() {
		loopErr <- r.loop.Run(runCtx)
	}()

	log.Printf("worker server listening at %v", r.listener.Addr())

	select {
	case <-ctx.Done():
		cancel()
		r.shutdownGRPC()
		loopRunErr := <-loopErr
		grpcRunErr := <-serveErr
		return firstNonNilErr(
			normalizeWorkerLoopErr(loopRunErr),
			normalizeWorkerServeErr(grpcRunErr),
		)
	case err := <-loopErr:
		cancel()
		r.shutdownGRPC()
		grpcRunErr := <-serveErr
		if loopRunErr := normalizeWorkerLoopErr(err); loopRunErr != nil {
			return loopRunErr
		}
		return normalizeWorkerServeErr(grpcRunErr)
	case err := <-serveErr:
		cancel()
		r.shutdownGRPC()
		loopRunErr := <-loopErr
		if grpcRunErr := normalizeWorkerServeErr(err); grpcRunErr != nil {
			if workerErr := normalizeWorkerLoopErr(loopRunErr); workerErr != nil {
				return firstNonNilErr(grpcRunErr, workerErr)
			}
			return grpcRunErr
		}
		return normalizeWorkerLoopErr(loopRunErr)
	}
}

// Close releases runtime resources.
func (r *Runtime) Close() {
	if r == nil {
		return
	}
	r.closeOnce.Do(func() {
		if r.health != nil {
			r.health.Shutdown()
		}
		if r.grpcServer != nil {
			r.grpcServer.Stop()
		}
		if r.listener != nil {
			if err := r.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				log.Printf("close worker listener: %v", err)
			}
		}
		closeManagedConn(r.socialMc, "social")
		closeManagedConn(r.notificationsMc, "notifications")
		closeManagedConn(r.authMc, "auth")
		if r.store != nil {
			if err := r.store.Close(); err != nil {
				log.Printf("close worker sqlite store: %v", err)
			}
		}
	})
}

func normalizeRuntimeConfig(cfg RuntimeConfig) (RuntimeConfig, error) {
	cfg.AuthAddr = strings.TrimSpace(cfg.AuthAddr)
	cfg.NotificationsAddr = strings.TrimSpace(cfg.NotificationsAddr)
	cfg.SocialAddr = strings.TrimSpace(cfg.SocialAddr)
	if cfg.AuthAddr == "" {
		return RuntimeConfig{}, fmt.Errorf("auth address is required")
	}
	if cfg.NotificationsAddr == "" {
		return RuntimeConfig{}, fmt.Errorf("notifications address is required")
	}
	if cfg.SocialAddr == "" {
		return RuntimeConfig{}, fmt.Errorf("social address is required")
	}
	if cfg.Port <= 0 {
		cfg.Port = defaultWorkerPort
	}
	cfg.DBPath = strings.TrimSpace(cfg.DBPath)
	if cfg.DBPath == "" {
		cfg.DBPath = defaultWorkerDB
	}
	return cfg, nil
}

func (r *Runtime) shutdownGRPC() {
	if r == nil {
		return
	}
	if r.health != nil {
		r.health.Shutdown()
	}
	if r.grpcServer != nil {
		r.grpcServer.GracefulStop()
	}
}

func normalizeWorkerServeErr(err error) error {
	if err == nil || errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return fmt.Errorf("serve gRPC: %w", err)
}

func normalizeWorkerLoopErr(err error) error {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return fmt.Errorf("run worker loop: %w", err)
}

func closeManagedConn(mc *platformgrpc.ManagedConn, name string) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		log.Printf("close worker %s managed conn: %v", name, err)
	}
}

func firstNonNilErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func fanoutEventHandlers(handlers ...EventHandler) EventHandler {
	filtered := make([]EventHandler, 0, len(handlers))
	for _, handler := range handlers {
		if handler == nil {
			continue
		}
		filtered = append(filtered, handler)
	}
	if len(filtered) == 0 {
		return nil
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return EventHandlerFunc(func(ctx context.Context, event *authv1.IntegrationOutboxEvent) error {
		for _, handler := range filtered {
			if err := handler.Handle(ctx, event); err != nil {
				return err
			}
		}
		return nil
	})
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
