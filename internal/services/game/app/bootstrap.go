package server

import (
	"context"
	"fmt"
	"log"
	"net"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/interceptors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartservice "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/random"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	storagesqlite "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// serverBootstrap configures each startup phase for the game server.
type serverBootstrap struct {
	config serverBootstrapConfig
}

// serverBootstrapConfig defines per-phase seams and is intentionally internal.
type serverBootstrapConfig struct {
	loadEnv                         func() serverEnv
	listen                          func(network, address string) (net.Listener, error)
	openStorageBundle               storageBundleOpener
	configureDomain                 func(serverEnv, *gamegrpc.Stores, engine.Registries) error
	buildSystemRegistry             func() (*bridge.MetadataRegistry, error)
	validateSystemRegistration      func([]module.Module, *bridge.MetadataRegistry, *bridge.AdapterRegistry) error
	dialAuthGRPC                    func(context.Context, string) (authGRPCClients, error)
	newGRPCServer                   func(*storageBundle) *grpc.Server
	newHealthServer                 func() *health.Server
	resolveProjectionApplyModes     func(serverEnv) (bool, bool, string, error)
	buildProjectionRegistries       func(engine.Registries, *bridge.AdapterRegistry) (*event.Registry, error)
	buildProjectionApplyOutboxApply func(*storagesqlite.Store, *event.Registry) func(context.Context, event.Event) error
}

// storageBundleOpener creates startup storage bundles for the server.
// This interface keeps storage construction injectable for tests and future
// non-SQLite implementations.
type storageBundleOpener interface {
	Open(serverEnv, *event.Registry) (*storageBundle, error)
}

// storageBundleOpenerFunc adapts an existing function to storageBundleOpener.
type storageBundleOpenerFunc func(serverEnv, *event.Registry) (*storageBundle, error)

// Open satisfies storageBundleOpener.
func (openFn storageBundleOpenerFunc) Open(env serverEnv, eventRegistry *event.Registry) (*storageBundle, error) {
	return openFn(env, eventRegistry)
}

func newServerBootstrap() *serverBootstrap {
	return newServerBootstrapWithConfig(serverBootstrapConfig{})
}

func newServerBootstrapWithConfig(cfg serverBootstrapConfig) *serverBootstrap {
	cfg = normalizeServerBootstrapConfig(cfg)
	return &serverBootstrap{config: cfg}
}

func normalizeServerBootstrapConfig(cfg serverBootstrapConfig) serverBootstrapConfig {
	if cfg.loadEnv == nil {
		cfg.loadEnv = loadServerEnv
	}
	if cfg.listen == nil {
		cfg.listen = net.Listen
	}
	if cfg.openStorageBundle == nil {
		cfg.openStorageBundle = storageBundleOpenerFunc(func(env serverEnv, eventRegistry *event.Registry) (*storageBundle, error) {
			return openStorageBundle(env, eventRegistry)
		})
	}
	if cfg.configureDomain == nil {
		cfg.configureDomain = configureDomain
	}
	if cfg.buildSystemRegistry == nil {
		cfg.buildSystemRegistry = buildSystemRegistry
	}
	if cfg.validateSystemRegistration == nil {
		cfg.validateSystemRegistration = validateSystemRegistrationParity
	}
	if cfg.dialAuthGRPC == nil {
		cfg.dialAuthGRPC = dialAuthGRPC
	}
	if cfg.newGRPCServer == nil {
		cfg.newGRPCServer = func(bundle *storageBundle) *grpc.Server {
			return grpc.NewServer(
				grpc.StatsHandler(otelgrpc.NewServerHandler()),
				grpc.ChainUnaryInterceptor(
					grpcmeta.UnaryServerInterceptor(nil),
					interceptors.AuditInterceptor(bundle.events),
					interceptors.SessionLockInterceptor(bundle.projections),
				),
				grpc.ChainStreamInterceptor(
					grpcmeta.StreamServerInterceptor(nil),
				),
			)
		}
	}
	if cfg.newHealthServer == nil {
		cfg.newHealthServer = health.NewServer
	}
	if cfg.resolveProjectionApplyModes == nil {
		cfg.resolveProjectionApplyModes = resolveProjectionApplyOutboxModes
	}
	if cfg.buildProjectionRegistries == nil {
		cfg.buildProjectionRegistries = buildProjectionRegistries
	}
	if cfg.buildProjectionApplyOutboxApply == nil {
		cfg.buildProjectionApplyOutboxApply = buildProjectionApplyOutboxApply
	}
	return cfg
}

// NewWithAddr builds a game server using named startup phases.
// It preserves the previous startup behavior while improving testability through dependency seams.
func (b *serverBootstrap) NewWithAddr(addr string) (server *Server, err error) {
	srvEnv := b.config.loadEnv()

	registries, err := engine.BuildRegistries(registeredSystemModules()...)
	if err != nil {
		return nil, fmt.Errorf("build registries: %w", err)
	}

	listener, err := b.config.listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}
	defer func() {
		if err == nil {
			return
		}
		_ = listener.Close()
	}()

	bundle, err := b.config.openStorageBundle.Open(srvEnv, registries.Events)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			return
		}
		bundle.Close()
	}()

	stores := gamegrpc.Stores{
		Campaign:           bundle.projections,
		Participant:        bundle.projections,
		ClaimIndex:         bundle.projections,
		Invite:             bundle.projections,
		Character:          bundle.projections,
		SystemStores:       systemmanifest.ProjectionStores{Daggerheart: bundle.projections},
		Session:            bundle.projections,
		SessionGate:        bundle.projections,
		SessionSpotlight:   bundle.projections,
		Event:              bundle.events,
		Audit:              bundle.events,
		Statistics:         bundle.projections,
		Snapshot:           bundle.projections,
		CampaignFork:       bundle.projections,
		DaggerheartContent: bundle.content,
	}
	if err := stores.Validate(); err != nil {
		return nil, fmt.Errorf("validate stores: %w", err)
	}
	if err := b.config.configureDomain(srvEnv, &stores, registries); err != nil {
		return nil, fmt.Errorf("configure domain: %w", err)
	}
	systemRegistry, err := b.config.buildSystemRegistry()
	if err != nil {
		return nil, fmt.Errorf("build system registry: %w", err)
	}
	applier, err := stores.TryApplier()
	if err != nil {
		return nil, fmt.Errorf("build projection applier: %w", err)
	}
	repairProjectionGaps(bundle, applier)
	if err := b.config.validateSystemRegistration(registeredSystemModules(), systemRegistry, applier.Adapters); err != nil {
		return nil, fmt.Errorf("validate system parity: %w", err)
	}
	authClients, err := b.config.dialAuthGRPC(context.Background(), srvEnv.AuthAddr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			return
		}
		if authClients.conn != nil {
			_ = authClients.conn.Close()
		}
	}()

	grpcServer := b.config.newGRPCServer(bundle)
	healthServer := b.config.newHealthServer()
	if err := b.registerServices(grpcServer, healthServer, stores, bundle, authClients.authClient, systemRegistry); err != nil {
		return nil, err
	}

	enableApplyWorker, enableShadowWorker, projectionApplyMode, err := b.config.resolveProjectionApplyModes(srvEnv)
	if err != nil {
		return nil, fmt.Errorf("resolve projection apply outbox modes: %w", err)
	}
	gamegrpc.SetInlineProjectionApplyEnabled(projectionApplyMode != projectionApplyModeOutboxApplyOnly)
	gamegrpc.SetCompatibilityAppendEnabled(srvEnv.CompatibilityAppendEnabled)
	daggerheartservice.SetInlineProjectionApplyEnabled(projectionApplyMode != projectionApplyModeOutboxApplyOnly)
	log.Printf("projection apply mode = %s", projectionApplyMode)

	projectionRegistries, err := b.config.buildProjectionRegistries(registries, applier.Adapters)
	if err != nil {
		return nil, fmt.Errorf("build projection registries: %w", err)
	}
	gamegrpc.SetIntentFilter(projectionRegistries)
	daggerheartservice.SetIntentFilter(projectionRegistries)

	server = &Server{
		listener:                                 listener,
		grpcServer:                               grpcServer,
		health:                                   healthServer,
		stores:                                   bundle,
		authConn:                                 authClients.conn,
		projectionApplyOutboxWorkerEnabled:       enableApplyWorker,
		projectionApplyOutboxApply:               b.config.buildProjectionApplyOutboxApply(bundle.projections, projectionRegistries),
		projectionApplyOutboxShadowWorkerEnabled: enableShadowWorker,
	}

	listener = nil
	bundle = nil
	authClients.conn = nil
	return server, nil
}

func (b *serverBootstrap) registerServices(
	grpcServer *grpc.Server, healthServer *health.Server,
	stores gamegrpc.Stores,
	bundle *storageBundle, authClient authv1.AuthServiceClient,
	systemRegistry *bridge.MetadataRegistry,
) error {
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
		return fmt.Errorf("create daggerheart service: %w", err)
	}
	contentService, err := daggerheartservice.NewDaggerheartContentService(daggerheartStores)
	if err != nil {
		return fmt.Errorf("create daggerheart content service: %w", err)
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
	authorizationService := gamegrpc.NewAuthorizationService(stores)

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
	statev1.RegisterAuthorizationServiceServer(grpcServer, authorizationService)
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
	healthServer.SetServingStatus("game.v1.AuthorizationService", grpc_health_v1.HealthCheckResponse_SERVING)
	return nil
}
