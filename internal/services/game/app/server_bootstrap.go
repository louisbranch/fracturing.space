package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
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
	daggerheartService, err := daggerheartservice.NewDaggerheartService(daggerheartStores, random.NewSeed)
	if err != nil {
		_ = listener.Close()
		bundle.Close()
		return nil, fmt.Errorf("create daggerheart service: %w", err)
	}
	contentService, err := daggerheartservice.NewDaggerheartContentService(daggerheartStores)
	if err != nil {
		_ = listener.Close()
		bundle.Close()
		return nil, fmt.Errorf("create daggerheart content service: %w", err)
	}
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

	projectionRegistries, err := engine.BuildRegistries(registeredSystemModules()...)
	if err != nil {
		_ = listener.Close()
		bundle.Close()
		return nil, fmt.Errorf("build projection registries: %w", err)
	}

	return &Server{
		listener:                                 listener,
		grpcServer:                               grpcServer,
		health:                                   healthServer,
		stores:                                   bundle,
		authConn:                                 authConn,
		projectionApplyOutboxWorkerEnabled:       enableApplyWorker,
		projectionApplyOutboxApply:               buildProjectionApplyOutboxApply(bundle.projections, projectionRegistries.Events),
		projectionApplyOutboxShadowWorkerEnabled: enableShadowWorker,
	}, nil
}

func buildProjectionApplyOutboxApply(projectionStore *storagesqlite.Store, eventRegistry *event.Registry) func(context.Context, event.Event) error {
	if projectionStore == nil {
		return nil
	}
	return func(ctx context.Context, evt event.Event) error {
		_, err := projectionStore.ApplyProjectionEventExactlyOnce(
			ctx,
			evt,
			func(applyCtx context.Context, applyEvt event.Event, txStore *storagesqlite.Store) error {
				txApplier := projection.Applier{
					Events:           eventRegistry,
					Campaign:         txStore,
					Character:        txStore,
					CampaignFork:     txStore,
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
