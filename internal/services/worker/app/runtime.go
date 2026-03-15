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

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	gameintegration "github.com/louisbranch/fracturing.space/internal/services/game/integration"
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
	AIAddr            string
	GameAddr          string
	NotificationsAddr string
	SocialAddr        string
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

const socialDirectoryBackfillPageSize = 50

type workerLoop interface {
	Run(ctx context.Context) error
}

type authDirectoryBootstrapClient interface {
	ListUsers(ctx context.Context, in *authv1.ListUsersRequest, opts ...grpc.CallOption) (*authv1.ListUsersResponse, error)
}

type socialDirectoryBootstrapClient interface {
	SyncDirectoryUser(ctx context.Context, in *socialv1.SyncDirectoryUserRequest, opts ...grpc.CallOption) (*socialv1.SyncDirectoryUserResponse, error)
}

// Runtime wires worker transport, dependency clients, and loop execution.
type Runtime struct {
	listener   net.Listener
	grpcServer *grpc.Server
	health     *health.Server

	loop workerLoop

	store           *workersqlite.Store
	authMc          *platformgrpc.ManagedConn
	aiMc            *platformgrpc.ManagedConn
	gameMc          *platformgrpc.ManagedConn
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
		closeWorkerStore(workerStore)
		return nil, fmt.Errorf("worker: managed conn auth: %w", err)
	}

	aiMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "ai",
		Addr: normalized.AIAddr,
		Mode: platformgrpc.ModeOptional,
		Logf: logf,
	})
	if err != nil {
		closeManagedConn(authMc, "auth")
		closeWorkerStore(workerStore)
		return nil, fmt.Errorf("worker: managed conn ai: %w", err)
	}

	gameMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "game",
		Addr: normalized.GameAddr,
		Mode: platformgrpc.ModeOptional,
		Logf: logf,
	})
	if err != nil {
		closeManagedConn(aiMc, "ai")
		closeManagedConn(authMc, "auth")
		closeWorkerStore(workerStore)
		return nil, fmt.Errorf("worker: managed conn game: %w", err)
	}

	notificationsMc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "notifications",
		Addr: normalized.NotificationsAddr,
		Mode: platformgrpc.ModeOptional,
		Logf: logf,
	})
	if err != nil {
		closeManagedConn(gameMc, "game")
		closeManagedConn(aiMc, "ai")
		closeManagedConn(authMc, "auth")
		closeWorkerStore(workerStore)
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
		closeManagedConn(gameMc, "game")
		closeManagedConn(aiMc, "ai")
		closeManagedConn(authMc, "auth")
		closeWorkerStore(workerStore)
		return nil, fmt.Errorf("worker: managed conn social: %w", err)
	}

	authClient := authv1.NewAuthServiceClient(authMc.Conn())
	socialClient := socialv1.NewSocialServiceClient(socialMc.Conn())
	if err := syncSocialUserDirectory(ctx, authClient, socialClient); err != nil {
		closeManagedConn(socialMc, "social")
		closeManagedConn(notificationsMc, "notifications")
		closeManagedConn(gameMc, "game")
		closeManagedConn(aiMc, "ai")
		closeManagedConn(authMc, "auth")
		closeWorkerStore(workerStore)
		return nil, fmt.Errorf("backfill social user directory: %w", err)
	}

	aiCampaignClient := aiv1.NewCampaignOrchestrationServiceClient(aiMc.Conn())
	gameInviteClient := gamev1.NewInviteServiceClient(gameMc.Conn())
	gameCampaignClient := gamev1.NewCampaignServiceClient(gameMc.Conn())
	gameCampaignAIClient := gamev1.NewCampaignAIServiceClient(gameMc.Conn())
	gameParticipantClient := gamev1.NewParticipantServiceClient(gameMc.Conn())
	gameCampaignAIOrchestrationClient := gamev1.NewCampaignAIOrchestrationServiceClient(gameMc.Conn())
	notificationsClient := notificationsv1.NewNotificationServiceClient(notificationsMc.Conn())

	signupProfileHandler := workerdomain.NewSignupSocialProfileHandler(socialClient)
	signupDirectoryHandler := workerdomain.NewSignupSocialDirectoryHandler(socialClient)
	signupHandler := fanoutEventHandlers(signupDirectoryHandler, signupProfileHandler)
	inviteCreatedHandler := workerdomain.NewInviteCreatedNotificationHandler(gameInviteClient, gameCampaignClient, gameParticipantClient, authClient, notificationsClient)
	inviteAcceptedHandler := workerdomain.NewInviteAcceptedNotificationHandler(gameInviteClient, gameCampaignClient, gameParticipantClient, authClient, notificationsClient)
	inviteDeclinedHandler := workerdomain.NewInviteDeclinedNotificationHandler(gameInviteClient, gameCampaignClient, gameParticipantClient, authClient, notificationsClient)
	aiGMTurnRequestedHandler := workerdomain.NewAIGMTurnRequestedHandler(gameCampaignAIOrchestrationClient, gameCampaignAIClient, aiCampaignClient)

	loopConfig := Config{
		Consumer:      normalized.Consumer,
		PollInterval:  normalized.PollInterval,
		LeaseTTL:      normalized.LeaseTTL,
		MaxAttempts:   normalized.MaxAttempts,
		RetryBackoff:  normalized.RetryBackoff,
		RetryMaxDelay: normalized.RetryMaxDelay,
	}
	normalizedLoopConfig := loopConfig.normalized()
	recorder := newAttemptStoreRecorder(workerStore, normalizedLoopConfig.Consumer)

	authLoop := New(
		"auth",
		newAuthOutboxClientAdapter(authClient),
		recorder,
		map[string]EventHandler{
			"auth.signup_completed": signupHandler,
		},
		normalizedLoopConfig,
		nil,
	)
	gameLoop := New(
		"game",
		newGameOutboxClientAdapter(gamev1.NewIntegrationServiceClient(gameMc.Conn())),
		recorder,
		map[string]EventHandler{
			gameintegration.InviteNotificationCreatedOutboxEventType:  inviteCreatedHandler,
			gameintegration.InviteNotificationClaimedOutboxEventType:  inviteAcceptedHandler,
			gameintegration.InviteNotificationDeclinedOutboxEventType: inviteDeclinedHandler,
			gameintegration.AIGMTurnRequestedOutboxEventType:          aiGMTurnRequestedHandler,
		},
		normalizedLoopConfig,
		nil,
	)

	listener, err := listenTCP("tcp", fmt.Sprintf(":%d", normalized.Port))
	if err != nil {
		closeManagedConn(socialMc, "social")
		closeManagedConn(notificationsMc, "notifications")
		closeManagedConn(gameMc, "game")
		closeManagedConn(aiMc, "ai")
		closeManagedConn(authMc, "auth")
		closeWorkerStore(workerStore)
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
		loop:            parallelLoop{loops: []workerLoop{authLoop, gameLoop}},
		store:           workerStore,
		authMc:          authMc,
		aiMc:            aiMc,
		gameMc:          gameMc,
		notificationsMc: notificationsMc,
		socialMc:        socialMc,
	}, nil
}

func syncSocialUserDirectory(ctx context.Context, authClient authDirectoryBootstrapClient, socialClient socialDirectoryBootstrapClient) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if authClient == nil {
		return errors.New("auth client is required")
	}
	if socialClient == nil {
		return errors.New("social client is required")
	}

	pageToken := ""
	for {
		resp, err := authClient.ListUsers(ctx, &authv1.ListUsersRequest{
			PageSize:  socialDirectoryBackfillPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			return err
		}
		for _, user := range resp.GetUsers() {
			if user == nil {
				continue
			}
			userID := strings.TrimSpace(user.GetId())
			username := strings.TrimSpace(user.GetUsername())
			if userID == "" || username == "" {
				return fmt.Errorf("list users returned blank directory fields")
			}
			if _, err := socialClient.SyncDirectoryUser(ctx, &socialv1.SyncDirectoryUserRequest{
				UserId:   userID,
				Username: username,
			}); err != nil {
				return err
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			return nil
		}
	}
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
		return firstNonNilErr(normalizeWorkerLoopErr(loopRunErr), normalizeWorkerServeErr(grpcRunErr))
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
		closeManagedConn(r.gameMc, "game")
		closeManagedConn(r.aiMc, "ai")
		closeManagedConn(r.authMc, "auth")
		closeWorkerStore(r.store)
	})
}

func normalizeRuntimeConfig(cfg RuntimeConfig) (RuntimeConfig, error) {
	cfg.AuthAddr = strings.TrimSpace(cfg.AuthAddr)
	cfg.AIAddr = strings.TrimSpace(cfg.AIAddr)
	cfg.GameAddr = strings.TrimSpace(cfg.GameAddr)
	cfg.NotificationsAddr = strings.TrimSpace(cfg.NotificationsAddr)
	cfg.SocialAddr = strings.TrimSpace(cfg.SocialAddr)
	if cfg.AuthAddr == "" {
		return RuntimeConfig{}, fmt.Errorf("auth address is required")
	}
	if cfg.AIAddr == "" {
		return RuntimeConfig{}, fmt.Errorf("ai address is required")
	}
	if cfg.GameAddr == "" {
		return RuntimeConfig{}, fmt.Errorf("game address is required")
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

func closeWorkerStore(store *workersqlite.Store) {
	if store == nil {
		return
	}
	if err := store.Close(); err != nil {
		log.Printf("close worker sqlite store: %v", err)
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
	return EventHandlerFunc(func(ctx context.Context, event workerdomain.OutboxEvent) error {
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

func canonicalOutcomeValue(outcome workerdomain.AckOutcome) string {
	return outcome.String()
}
