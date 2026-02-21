package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/config"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/interceptors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartservice "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// serverEnv holds env-parsed configuration for the game server.
type serverEnv struct {
	AuthAddr                                 string `env:"FRACTURING_SPACE_AUTH_ADDR"                                 envDefault:"localhost:8083"`
	EventsDBPath                             string `env:"FRACTURING_SPACE_GAME_EVENTS_DB_PATH"`
	ProjectionsDBPath                        string `env:"FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH"`
	ContentDBPath                            string `env:"FRACTURING_SPACE_GAME_CONTENT_DB_PATH"`
	DomainEnabled                            bool   `env:"FRACTURING_SPACE_GAME_DOMAIN_ENABLED"                       envDefault:"true"`
	CompatibilityAppendEnabled               bool   `env:"FRACTURING_SPACE_GAME_COMPATIBILITY_APPEND_ENABLED"         envDefault:"false"`
	ProjectionApplyOutboxEnabled             bool   `env:"FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_ENABLED"     envDefault:"false"`
	ProjectionApplyOutboxShadowWorkerEnabled bool   `env:"FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_SHADOW_WORKER_ENABLED" envDefault:"false"`
	ProjectionApplyOutboxWorkerEnabled       bool   `env:"FRACTURING_SPACE_GAME_PROJECTION_APPLY_OUTBOX_WORKER_ENABLED" envDefault:"false"`
}

const (
	projectionApplyModeInlineApplyOnly = "inline_apply_only"
	projectionApplyModeOutboxApplyOnly = "outbox_apply_only"
	projectionApplyModeShadowOnly      = "shadow_only"
)

func loadServerEnv() serverEnv {
	var cfg serverEnv
	_ = config.ParseEnv(&cfg)
	if cfg.EventsDBPath == "" {
		cfg.EventsDBPath = filepath.Join("data", "game-events.db")
	}
	if cfg.ProjectionsDBPath == "" {
		cfg.ProjectionsDBPath = filepath.Join("data", "game-projections.db")
	}
	if cfg.ContentDBPath == "" {
		cfg.ContentDBPath = filepath.Join("data", "game-content.db")
	}
	return cfg
}

func resolveProjectionApplyOutboxModes(srvEnv serverEnv) (bool, bool, string, error) {
	if !srvEnv.ProjectionApplyOutboxEnabled {
		if srvEnv.ProjectionApplyOutboxWorkerEnabled {
			return false, false, "", errors.New("projection apply outbox worker requested without outbox enabled")
		}
		if srvEnv.ProjectionApplyOutboxShadowWorkerEnabled {
			return false, false, "", errors.New("projection apply outbox shadow worker requested without outbox enabled")
		}
		return false, false, projectionApplyModeInlineApplyOnly, nil
	}

	if srvEnv.ProjectionApplyOutboxWorkerEnabled && srvEnv.ProjectionApplyOutboxShadowWorkerEnabled {
		return false, false, "", errors.New("projection apply outbox cannot enable both apply and shadow workers")
	}
	if srvEnv.ProjectionApplyOutboxWorkerEnabled {
		return true, false, projectionApplyModeOutboxApplyOnly, nil
	}
	if srvEnv.ProjectionApplyOutboxShadowWorkerEnabled {
		return false, true, projectionApplyModeShadowOnly, nil
	}
	return false, false, projectionApplyModeInlineApplyOnly, nil
}

// Server hosts the game service.
type Server struct {
	listener                                 net.Listener
	grpcServer                               *grpc.Server
	health                                   *health.Server
	stores                                   *storageBundle
	authConn                                 *grpc.ClientConn
	projectionApplyOutboxWorkerEnabled       bool
	projectionApplyOutboxApply               func(context.Context, event.Event) error
	projectionApplyOutboxShadowWorkerEnabled bool
}

type authGRPCClients struct {
	conn       *grpc.ClientConn
	authClient authv1.AuthServiceClient
}

// projectionApplyOutboxShadowProcessor drains queue rows for environments where the
// main apply worker is intentionally delayed or disabled.
type projectionApplyOutboxShadowProcessor interface {
	ProcessProjectionApplyOutboxShadow(context.Context, time.Time, int) (int, error)
}

// projectionApplyOutboxProcessor is responsible for applying queued events to projections.
//
// It keeps event ingestion and projection side effects separated from request path
// responsiveness while still converging read models in the background.
type projectionApplyOutboxProcessor interface {
	ProcessProjectionApplyOutbox(context.Context, time.Time, int, func(context.Context, event.Event) error) (int, error)
}

// Projection worker defaults balance recovery speed versus DB churn.
const (
	projectionApplyOutboxWorkerInterval       = 2 * time.Second
	projectionApplyOutboxWorkerBatch          = 64
	projectionApplyOutboxShadowWorkerInterval = 2 * time.Second
	projectionApplyOutboxShadowWorkerBatch    = 64
)

// storageBundle groups the three SQLite stores and manages their lifecycle.
//
// Events are the source of truth, projections feed APIs, and content stores
// enrich projection reads for system-specific metadata.
type storageBundle struct {
	events      *storagesqlite.Store
	projections *storagesqlite.Store
	content     *storagesqlite.Store
}

// Close closes all stores in the bundle, logging any errors.
func (b *storageBundle) Close() {
	if b == nil {
		return
	}
	if b.events != nil {
		if err := b.events.Close(); err != nil {
			log.Printf("close event store: %v", err)
		}
	}
	if b.projections != nil {
		if err := b.projections.Close(); err != nil {
			log.Printf("close projection store: %v", err)
		}
	}
	if b.content != nil {
		if err := b.content.Close(); err != nil {
			log.Printf("close content store: %v", err)
		}
	}
}

// New creates a configured game server listening on the provided port.
func New(port int) (*Server, error) {
	return NewWithAddr(fmt.Sprintf(":%d", port))
}

// NewWithAddr creates a configured game server listening on the provided address.
func NewWithAddr(addr string) (*Server, error) {
	srvEnv := loadServerEnv()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}
	bundle, err := openStorageBundle(srvEnv)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	stores := gamegrpc.Stores{
		Campaign:           bundle.projections,
		Participant:        bundle.projections,
		ClaimIndex:         bundle.projections,
		Invite:             bundle.projections,
		Character:          bundle.projections,
		Daggerheart:        bundle.projections,
		Session:            bundle.projections,
		SessionGate:        bundle.projections,
		SessionSpotlight:   bundle.projections,
		Event:              bundle.events,
		Telemetry:          bundle.events,
		Statistics:         bundle.projections,
		Snapshot:           bundle.projections,
		CampaignFork:       bundle.projections,
		DaggerheartContent: bundle.content,
	}
	if err := stores.Validate(); err != nil {
		_ = listener.Close()
		bundle.Close()
		return nil, fmt.Errorf("validate stores: %w", err)
	}
	if err := configureDomain(srvEnv, &stores); err != nil {
		_ = listener.Close()
		bundle.Close()
		return nil, fmt.Errorf("configure domain: %w", err)
	}
	systemRegistry, err := buildSystemRegistry()
	if err != nil {
		_ = listener.Close()
		bundle.Close()
		return nil, fmt.Errorf("build system registry: %w", err)
	}
	applier := stores.Applier()
	if err := validateSystemRegistrationParity(registeredSystemModules(), systemRegistry, applier.Adapters); err != nil {
		_ = listener.Close()
		bundle.Close()
		return nil, fmt.Errorf("validate system parity: %w", err)
	}

	authClients, err := dialAuthGRPC(context.Background(), srvEnv.AuthAddr)
	if err != nil {
		_ = listener.Close()
		bundle.Close()
		return nil, err
	}
	authConn := authClients.conn
	authClient := authClients.authClient

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			grpcmeta.UnaryServerInterceptor(nil),
			interceptors.TelemetryInterceptor(bundle.events),
			interceptors.SessionLockInterceptor(bundle.projections),
		),
		grpc.ChainStreamInterceptor(
			grpcmeta.StreamServerInterceptor(nil),
		),
	)
	daggerheartStores := daggerheartservice.Stores{
		Campaign:           bundle.projections,
		Character:          bundle.projections,
		Session:            bundle.projections,
		SessionGate:        bundle.projections,
		SessionSpotlight:   bundle.projections,
		Daggerheart:        bundle.projections,
		DaggerheartContent: bundle.content,
		Event:              bundle.events,
		Domain:             stores.Domain,
	}
	daggerheartService := daggerheartservice.NewDaggerheartService(daggerheartStores, random.NewSeed)
	contentService := daggerheartservice.NewDaggerheartContentService(daggerheartStores)
	campaignService := gamegrpc.NewCampaignServiceWithAuth(stores, authClient)
	participantService := gamegrpc.NewParticipantService(stores)
	inviteService := gamegrpc.NewInviteServiceWithAuth(stores, authClient)
	characterService := gamegrpc.NewCharacterService(stores)
	snapshotService := gamegrpc.NewSnapshotService(stores)
	sessionService := gamegrpc.NewSessionService(stores)
	forkService := gamegrpc.NewForkService(stores)
	eventService := gamegrpc.NewEventService(stores)
	statisticsService := gamegrpc.NewStatisticsService(stores)
	systemService := gamegrpc.NewSystemService(systemRegistry)
	healthServer := health.NewServer()
	daggerheartv1.RegisterDaggerheartServiceServer(grpcServer, daggerheartService)
	daggerheartv1.RegisterDaggerheartContentServiceServer(grpcServer, contentService)
	statev1.RegisterCampaignServiceServer(grpcServer, campaignService)
	statev1.RegisterParticipantServiceServer(grpcServer, participantService)
	statev1.RegisterInviteServiceServer(grpcServer, inviteService)
	statev1.RegisterCharacterServiceServer(grpcServer, characterService)
	statev1.RegisterSnapshotServiceServer(grpcServer, snapshotService)
	statev1.RegisterSessionServiceServer(grpcServer, sessionService)
	statev1.RegisterForkServiceServer(grpcServer, forkService)
	statev1.RegisterEventServiceServer(grpcServer, eventService)
	statev1.RegisterStatisticsServiceServer(grpcServer, statisticsService)
	statev1.RegisterSystemServiceServer(grpcServer, systemService)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("systems.daggerheart.v1.DaggerheartService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("systems.daggerheart.v1.DaggerheartContentService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.CampaignService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.ParticipantService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.InviteService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.CharacterService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.SnapshotService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.SessionService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.ForkService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.EventService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.StatisticsService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.SystemService", grpc_health_v1.HealthCheckResponse_SERVING)

	enableApplyWorker, enableShadowWorker, projectionApplyOutboxMode, err := resolveProjectionApplyOutboxModes(srvEnv)
	if err != nil {
		_ = listener.Close()
		bundle.Close()
		return nil, fmt.Errorf("resolve projection apply outbox modes: %w", err)
	}
	gamegrpc.SetInlineProjectionApplyEnabled(projectionApplyOutboxMode != projectionApplyModeOutboxApplyOnly)
	gamegrpc.SetCompatibilityAppendEnabled(srvEnv.CompatibilityAppendEnabled)
	daggerheartservice.SetInlineProjectionApplyEnabled(projectionApplyOutboxMode != projectionApplyModeOutboxApplyOnly)
	log.Printf("projection apply mode = %s", projectionApplyOutboxMode)

	return &Server{
		listener:                                 listener,
		grpcServer:                               grpcServer,
		health:                                   healthServer,
		stores:                                   bundle,
		authConn:                                 authConn,
		projectionApplyOutboxWorkerEnabled:       enableApplyWorker,
		projectionApplyOutboxApply:               buildProjectionApplyOutboxApply(bundle.projections),
		projectionApplyOutboxShadowWorkerEnabled: enableShadowWorker,
	}, nil
}

func buildProjectionApplyOutboxApply(projectionStore *storagesqlite.Store) func(context.Context, event.Event) error {
	if projectionStore == nil {
		return nil
	}
	return func(ctx context.Context, evt event.Event) error {
		_, err := projectionStore.ApplyProjectionEventExactlyOnce(
			ctx,
			evt,
			func(applyCtx context.Context, applyEvt event.Event, txStore *storagesqlite.Store) error {
				txApplier := projection.Applier{
					Campaign:         txStore,
					Character:        txStore,
					CampaignFork:     txStore,
					Daggerheart:      txStore,
					ClaimIndex:       txStore,
					Invite:           txStore,
					Participant:      txStore,
					Session:          txStore,
					SessionGate:      txStore,
					SessionSpotlight: txStore,
					Adapters:         systemmanifest.AdapterRegistry(systemmanifest.ProjectionStores{Daggerheart: txStore}),
				}
				return txApplier.Apply(applyCtx, applyEvt)
			},
		)
		return err
	}
}

func buildSystemRegistry() (*systems.Registry, error) {
	registry := systems.NewRegistry()
	for _, gameSystem := range registeredMetadataSystems() {
		if err := registry.Register(gameSystem); err != nil {
			return nil, fmt.Errorf("register system %s@%s: %w", gameSystem.ID(), gameSystem.Version(), err)
		}
	}
	return registry, nil
}

// closeResources releases every handle created at server startup.
// Addr returns the listener address for the game server.
func (s *Server) Addr() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Run creates and serves a game server until the context ends.
func Run(ctx context.Context, port int) error {
	grpcServer, err := New(port)
	if err != nil {
		return err
	}
	return grpcServer.Serve(ctx)
}

// RunWithAddr creates and serves a game server until the context ends.
func RunWithAddr(ctx context.Context, addr string) error {
	grpcServer, err := NewWithAddr(addr)
	if err != nil {
		return err
	}
	return grpcServer.Serve(ctx)
}

// Serve starts the game server and blocks until it stops or the context ends.
func (s *Server) Serve(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	defer s.closeResources()
	stopOutboxWorker := s.startProjectionApplyOutboxWorker(ctx)
	defer stopOutboxWorker()
	stopOutboxShadowWorker := s.startProjectionApplyOutboxShadowWorker(ctx)
	defer stopOutboxShadowWorker()

	log.Printf("game server listening at %v", s.listener.Addr())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.grpcServer.Serve(s.listener)
	}()

	handleErr := func(err error) error {
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve gRPC: %w", err)
	}

	select {
	case <-ctx.Done():
		if s.health != nil {
			s.health.Shutdown()
		}
		s.grpcServer.GracefulStop()
		err := <-serveErr
		return handleErr(err)
	case err := <-serveErr:
		return handleErr(err)
	}
}

// startProjectionApplyOutboxShadowWorker launches an optional background shadow worker.
//
// This keeps pending queue items progressing when projection updates are not
// processed inline.
func (s *Server) startProjectionApplyOutboxShadowWorker(ctx context.Context) func() {
	if s == nil || !s.projectionApplyOutboxShadowWorkerEnabled || s.stores == nil || s.stores.events == nil {
		return func() {}
	}

	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runProjectionApplyOutboxShadowWorker(
			workerCtx,
			s.stores.events,
			projectionApplyOutboxShadowWorkerInterval,
			projectionApplyOutboxShadowWorkerBatch,
			time.Now,
			log.Printf,
		)
	}()

	return func() {
		cancel()
		<-done
	}
}

// startProjectionApplyOutboxWorker launches an optional background projection worker.
//
// The worker applies queued projection rows independently from request handling.
func (s *Server) startProjectionApplyOutboxWorker(ctx context.Context) func() {
	if s == nil || !s.projectionApplyOutboxWorkerEnabled || s.stores == nil || s.stores.events == nil || s.projectionApplyOutboxApply == nil {
		return func() {}
	}

	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runProjectionApplyOutboxWorker(
			workerCtx,
			s.stores.events,
			s.projectionApplyOutboxApply,
			projectionApplyOutboxWorkerInterval,
			projectionApplyOutboxWorkerBatch,
			time.Now,
			log.Printf,
		)
	}()

	return func() {
		cancel()
		<-done
	}
}

// runProjectionApplyOutboxShadowWorker drains projection outbox shadow entries.
//
// It is intentionally lightweight: the purpose is progress cleanup, not full
// projection mutation.
func runProjectionApplyOutboxShadowWorker(
	ctx context.Context,
	processor projectionApplyOutboxShadowProcessor,
	interval time.Duration,
	limit int,
	now func() time.Time,
	logf func(string, ...any),
) {
	if processor == nil || interval <= 0 || limit <= 0 {
		return
	}
	if now == nil {
		now = time.Now
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}

	runPass := func() int {
		processed, err := processor.ProcessProjectionApplyOutboxShadow(ctx, now().UTC(), limit)
		if err != nil {
			logf("projection apply outbox shadow worker pass failed: %v", err)
			return 0
		}
		if processed > 0 {
			logf("projection apply outbox shadow worker observed %d rows", processed)
		}
		return processed
	}

	for {
		if runPass() < limit {
			break
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runPass()
		}
	}
}

// runProjectionApplyOutboxWorker drains projection outbox entries into projections.
//
// It loops in bounded batches until no rows remain, then waits for timer ticks.
func runProjectionApplyOutboxWorker(
	ctx context.Context,
	processor projectionApplyOutboxProcessor,
	apply func(context.Context, event.Event) error,
	interval time.Duration,
	limit int,
	now func() time.Time,
	logf func(string, ...any),
) {
	if processor == nil || apply == nil || interval <= 0 || limit <= 0 {
		return
	}
	if now == nil {
		now = time.Now
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}

	runPass := func() int {
		processed, err := processor.ProcessProjectionApplyOutbox(ctx, now().UTC(), limit, apply)
		if err != nil {
			logf("projection apply outbox worker pass failed: %v", err)
			return 0
		}
		if processed > 0 {
			logf("projection apply outbox worker applied %d rows", processed)
		}
		return processed
	}

	for {
		if runPass() < limit {
			break
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runPass()
		}
	}
}

// openStorageBundle opens events, projections, and content stores as a unit.
//
// Startup is all-or-nothing so no request path sees a partial storage graph.
func openStorageBundle(srvEnv serverEnv) (*storageBundle, error) {
	eventStore, err := openEventStore(srvEnv.EventsDBPath, srvEnv.ProjectionApplyOutboxEnabled)
	if err != nil {
		return nil, err
	}
	projStore, err := openProjectionStore(srvEnv.ProjectionsDBPath)
	if err != nil {
		_ = eventStore.Close()
		return nil, err
	}
	contentStore, err := openContentStore(srvEnv.ContentDBPath)
	if err != nil {
		_ = eventStore.Close()
		_ = projStore.Close()
		return nil, err
	}
	return &storageBundle{
		events:      eventStore,
		projections: projStore,
		content:     contentStore,
	}, nil
}

// dialAuthGRPC opens an authenticated gRPC client to auth service.
func dialAuthGRPC(ctx context.Context, authAddr string) (authGRPCClients, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	logf := func(format string, args ...any) {
		log.Printf("auth %s", fmt.Sprintf(format, args...))
	}
	conn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		authAddr,
		timeouts.GRPCDial,
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		var dialErr *platformgrpc.DialError
		if errors.As(err, &dialErr) {
			if dialErr.Stage == platformgrpc.DialStageHealth {
				return authGRPCClients{}, fmt.Errorf("auth gRPC health check failed for %s: %w", authAddr, dialErr.Err)
			}
			return authGRPCClients{}, fmt.Errorf("dial auth gRPC %s: %w", authAddr, dialErr.Err)
		}
		return authGRPCClients{}, fmt.Errorf("dial auth gRPC %s: %w", authAddr, err)
	}
	return authGRPCClients{
		conn:       conn,
		authClient: authv1.NewAuthServiceClient(conn),
	}, nil
}

// openEventStore opens the immutable event store and verifies chain integrity on boot.
func openEventStore(path string, projectionApplyOutboxEnabled bool) (*storagesqlite.Store, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	keyring, err := integrity.KeyringFromEnv()
	if err != nil {
		return nil, err
	}
	registries, err := engine.BuildRegistries(registeredSystemModules()...)
	if err != nil {
		return nil, fmt.Errorf("build registries: %w", err)
	}
	store, err := storagesqlite.OpenEvents(
		path,
		keyring,
		registries.Events,
		storagesqlite.WithProjectionApplyOutboxEnabled(projectionApplyOutboxEnabled),
	)
	if err != nil {
		return nil, fmt.Errorf("open events store: %w", err)
	}
	if err := store.VerifyEventIntegrity(context.Background()); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("verify event integrity: %w", err)
	}
	return store, nil
}

// openProjectionStore opens the materialized views database.
func openProjectionStore(path string) (*storagesqlite.Store, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	store, err := storagesqlite.OpenProjections(path)
	if err != nil {
		return nil, fmt.Errorf("open projections store: %w", err)
	}
	return store, nil
}

// openContentStore opens the content reference database.
func openContentStore(path string) (*storagesqlite.Store, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	store, err := storagesqlite.OpenContent(path)
	if err != nil {
		return nil, fmt.Errorf("open content store: %w", err)
	}
	return store, nil
}

// ensureDir creates parent paths for sqlite files so startup can create DB files.
func ensureDir(path string) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create storage dir: %w", err)
		}
	}
	return nil
}

// closeResources releases all server resources in shutdown order.
func (s *Server) closeResources() {
	if s == nil {
		return
	}
	s.stores.Close()
	if s.authConn != nil {
		if err := s.authConn.Close(); err != nil {
			log.Printf("close auth conn: %v", err)
		}
	}
}
