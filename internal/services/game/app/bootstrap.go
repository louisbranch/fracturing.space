package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
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
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
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
	dialSocialGRPC                  func(context.Context, string) (socialGRPCClients, error)
	dialAIGRPC                      func(context.Context, string) (aiGRPCClients, error)
	newGRPCServer                   func(*storageBundle, serverEnv) *grpc.Server
	newHealthServer                 func() *health.Server
	resolveProjectionApplyModes     func(serverEnv) (bool, bool, string, error)
	buildProjectionRegistries       func(engine.Registries, *bridge.AdapterRegistry) (*event.Registry, error)
	buildProjectionApplyOutboxApply func(*storagesqlite.Store, *event.Registry) func(context.Context, event.Event) error
}

// storageBundleOpener creates startup storage bundles for the server.
// This interface keeps storage construction injectable for tests and future
// non-SQLite implementations.
type storageBundleOpener interface {
	Open(context.Context, serverEnv, *event.Registry) (*storageBundle, error)
}

// storageBundleOpenerFunc adapts an existing function to storageBundleOpener.
type storageBundleOpenerFunc func(context.Context, serverEnv, *event.Registry) (*storageBundle, error)

// Open satisfies storageBundleOpener.
func (openFn storageBundleOpenerFunc) Open(ctx context.Context, env serverEnv, eventRegistry *event.Registry) (*storageBundle, error) {
	return openFn(ctx, env, eventRegistry)
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
		cfg.openStorageBundle = storageBundleOpenerFunc(func(ctx context.Context, env serverEnv, eventRegistry *event.Registry) (*storageBundle, error) {
			return openStorageBundle(ctx, env, eventRegistry)
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
	if cfg.dialSocialGRPC == nil {
		cfg.dialSocialGRPC = dialSocialGRPC
	}
	if cfg.dialAIGRPC == nil {
		cfg.dialAIGRPC = dialAIGRPC
	}
	if cfg.newGRPCServer == nil {
		cfg.newGRPCServer = func(bundle *storageBundle, srvEnv serverEnv) *grpc.Server {
			internalIdentity := interceptors.InternalServiceIdentityConfig{
				MethodPrefixes:    []string{"/game.v1.CampaignAIService/"},
				AllowedServiceIDs: parseInternalServiceAllowlist(srvEnv.InternalServiceAllowlist),
			}
			return grpc.NewServer(
				grpc.StatsHandler(otelgrpc.NewServerHandler()),
				grpc.ChainUnaryInterceptor(
					grpcmeta.UnaryServerInterceptor(nil),
					interceptors.InternalServiceIdentityUnaryInterceptor(internalIdentity),
					interceptors.AuditInterceptor(bundle.events),
					interceptors.SessionLockInterceptor(bundle.projections),
				),
				grpc.ChainStreamInterceptor(
					grpcmeta.StreamServerInterceptor(nil),
					interceptors.InternalServiceIdentityStreamInterceptor(internalIdentity),
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

func startupContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// NewWithAddr builds a game server using named startup phases.
// It preserves the previous startup behavior while improving testability through dependency seams.
func (b *serverBootstrap) NewWithAddr(ctx context.Context, addr string) (server *Server, err error) {
	startupCtx := startupContext(ctx)
	srvEnv := b.config.loadEnv()

	registries, err := engine.BuildRegistries(registeredSystemModules()...)
	if err != nil {
		return nil, wrapStartupError(startupPhaseRegistries, "build registries", err)
	}

	listener, err := b.config.listen("tcp", addr)
	if err != nil {
		return nil, wrapStartupError(startupPhaseNetwork, fmt.Sprintf("listen on %s", addr), err)
	}
	defer func() {
		if err == nil {
			return
		}
		_ = listener.Close()
	}()

	bundle, err := b.config.openStorageBundle.Open(startupCtx, srvEnv, registries.Events)
	if err != nil {
		return nil, wrapStartupError(startupPhaseStorage, "open storage bundle", err)
	}
	defer func() {
		if err == nil {
			return
		}
		bundle.Close()
	}()

	writeRuntime := gamegrpc.NewWriteRuntime()
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
		Scene:              bundle.projections,
		SceneCharacter:     bundle.projections,
		SceneGate:          bundle.projections,
		SceneSpotlight:     bundle.projections,
		Event:              bundle.events,
		Watermarks:         bundle.projections,
		Audit:              bundle.events,
		Statistics:         bundle.projections,
		Snapshot:           bundle.projections,
		CampaignFork:       bundle.projections,
		DaggerheartContent: bundle.content,
		WriteRuntime:       writeRuntime,
		Events:             registries.Events,
	}
	if err := b.config.configureDomain(srvEnv, &stores, registries); err != nil {
		return nil, wrapStartupError(startupPhaseDomain, "configure domain", err)
	}
	if err := stores.Validate(); err != nil {
		return nil, wrapStartupError(startupPhaseDomain, "validate stores", err)
	}

	systemRegistry, err := b.config.buildSystemRegistry()
	if err != nil {
		return nil, wrapStartupError(startupPhaseSystems, "build system registry", err)
	}

	applier, err := stores.TryApplier()
	if err != nil {
		return nil, wrapStartupError(startupPhaseDomain, "build projection applier", err)
	}
	if err := b.config.validateSystemRegistration(registeredSystemModules(), systemRegistry, applier.Adapters); err != nil {
		return nil, wrapStartupError(startupPhaseSystems, "validate system parity", err)
	}
	repairProjectionGaps(startupCtx, bundle, applier)

	authClients, err := b.config.dialAuthGRPC(startupCtx, srvEnv.AuthAddr)
	if err != nil {
		return nil, wrapStartupError(startupPhaseDependencies, "dial auth gRPC", err)
	}
	defer func() {
		if err == nil {
			return
		}
		if authClients.conn != nil {
			_ = authClients.conn.Close()
		}
	}()

	socialClients, socialErr := b.config.dialSocialGRPC(startupCtx, srvEnv.SocialAddr)
	if socialErr != nil {
		log.Printf("social client unavailable; participant pronouns fallback disabled: %v", socialErr)
		socialClients = socialGRPCClients{}
	}
	defer func() {
		if err == nil {
			return
		}
		if socialClients.conn != nil {
			_ = socialClients.conn.Close()
		}
	}()

	aiClients, aiErr := b.config.dialAIGRPC(startupCtx, srvEnv.AIAddr)
	if aiErr != nil {
		log.Printf("ai client unavailable; campaign ai binding operations will be unavailable: %v", aiErr)
		aiClients = aiGRPCClients{}
	}
	defer func() {
		if err == nil {
			return
		}
		if aiClients.conn != nil {
			_ = aiClients.conn.Close()
		}
	}()
	stores.Social = socialClients.socialClient

	grpcServer := b.config.newGRPCServer(bundle, srvEnv)
	healthServer := b.config.newHealthServer()
	sessionGrantConfig, err := aisessiongrant.LoadConfigFromEnv(time.Now)
	if err != nil {
		return nil, wrapStartupError(startupPhaseDependencies, "load ai session grant config", err)
	}
	if err := b.registerServices(
		grpcServer,
		healthServer,
		stores,
		bundle,
		authClients.authClient,
		aiClients.agentClient,
		systemRegistry,
		sessionGrantConfig,
	); err != nil {
		return nil, wrapStartupError(startupPhaseTransport, "register gRPC services", err)
	}

	enableApplyWorker, enableShadowWorker, projectionApplyMode, err := b.config.resolveProjectionApplyModes(srvEnv)
	if err != nil {
		return nil, wrapStartupError(startupPhaseRuntime, "resolve projection apply outbox modes", err)
	}
	writeRuntime.SetInlineApplyEnabled(projectionApplyMode != projectionApplyModeOutboxApplyOnly)
	log.Printf("projection apply mode = %s", projectionApplyMode)

	projectionRegistries, err := b.config.buildProjectionRegistries(registries, applier.Adapters)
	if err != nil {
		return nil, wrapStartupError(startupPhaseRuntime, "build projection registries", err)
	}
	writeRuntime.SetIntentFilter(projectionRegistries)

	statusConn, statusClient := dialStatusLenient(startupCtx, srvEnv.StatusAddr)
	defer func() {
		if err == nil {
			return
		}
		if statusConn != nil {
			_ = statusConn.Close()
		}
	}()
	catalogState := evaluateCatalogCapabilityState(startupCtx, bundle.content)
	statusReporter := initStatusReporter(
		statusClient,
		socialClients.socialClient != nil,
		aiClients.agentClient != nil,
		catalogState,
	)

	server = &Server{
		listener:                                 listener,
		grpcServer:                               grpcServer,
		health:                                   healthServer,
		stores:                                   bundle,
		authConn:                                 authClients.conn,
		socialConn:                               socialClients.conn,
		aiConn:                                   aiClients.conn,
		statusConn:                               statusConn,
		projectionApplyOutboxWorkerEnabled:       enableApplyWorker,
		projectionApplyOutboxApply:               b.config.buildProjectionApplyOutboxApply(bundle.projections, projectionRegistries),
		projectionApplyOutboxShadowWorkerEnabled: enableShadowWorker,
		statusReporter:                           statusReporter,
		catalogReadyAtStartup:                    catalogState.Ready,
	}

	listener = nil
	bundle = nil
	authClients.conn = nil
	socialClients.conn = nil
	aiClients.conn = nil
	statusConn = nil
	return server, nil
}

type grpcServiceDescriptor struct {
	healthService string
	register      func(*grpc.Server)
}

func (b *serverBootstrap) registerServices(
	grpcServer *grpc.Server,
	healthServer *health.Server,
	stores gamegrpc.Stores,
	bundle *storageBundle,
	authClient authv1.AuthServiceClient,
	aiAgentClient aiv1.AgentServiceClient,
	systemRegistry *bridge.MetadataRegistry,
	sessionGrantConfig aisessiongrant.Config,
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
		Watermarks:         bundle.projections,
		Domain:             stores.Domain,
		WriteRuntime:       stores.WriteRuntime,
	}
	daggerheartService, err := daggerheartservice.NewDaggerheartService(daggerheartStores, random.NewSeed)
	if err != nil {
		return fmt.Errorf("create daggerheart service: %w", err)
	}
	contentService, err := daggerheartservice.NewDaggerheartContentService(daggerheartStores)
	if err != nil {
		return fmt.Errorf("create daggerheart content service: %w", err)
	}
	campaignService := gamegrpc.NewCampaignServiceWithAuthAndAI(stores, authClient, aiAgentClient)
	participantService := gamegrpc.NewParticipantService(stores)
	inviteService := gamegrpc.NewInviteServiceWithAuth(stores, authClient)
	characterService := gamegrpc.NewCharacterService(stores)
	snapshotService := gamegrpc.NewSnapshotService(stores)
	sessionService := gamegrpc.NewSessionService(stores)
	sceneService := gamegrpc.NewSceneService(stores)
	forkService := gamegrpc.NewForkService(stores)
	eventService := gamegrpc.NewEventService(stores)
	statisticsService := gamegrpc.NewStatisticsService(stores)
	systemService := gamegrpc.NewSystemService(systemRegistry)
	authorizationService := gamegrpc.NewAuthorizationService(stores)
	campaignAIService := gamegrpc.NewCampaignAIService(stores, sessionGrantConfig)

	descriptors := []grpcServiceDescriptor{
		{
			healthService: "systems.daggerheart.v1.DaggerheartService",
			register: func(server *grpc.Server) {
				daggerheartv1.RegisterDaggerheartServiceServer(server, daggerheartService)
			},
		},
		{
			healthService: "systems.daggerheart.v1.DaggerheartContentService",
			register: func(server *grpc.Server) {
				daggerheartv1.RegisterDaggerheartContentServiceServer(server, contentService)
			},
		},
		{
			healthService: "game.v1.CampaignService",
			register: func(server *grpc.Server) {
				statev1.RegisterCampaignServiceServer(server, campaignService)
			},
		},
		{
			healthService: "game.v1.CampaignAIService",
			register: func(server *grpc.Server) {
				statev1.RegisterCampaignAIServiceServer(server, campaignAIService)
			},
		},
		{
			healthService: "game.v1.ParticipantService",
			register: func(server *grpc.Server) {
				statev1.RegisterParticipantServiceServer(server, participantService)
			},
		},
		{
			healthService: "game.v1.InviteService",
			register: func(server *grpc.Server) {
				statev1.RegisterInviteServiceServer(server, inviteService)
			},
		},
		{
			healthService: "game.v1.CharacterService",
			register: func(server *grpc.Server) {
				statev1.RegisterCharacterServiceServer(server, characterService)
			},
		},
		{
			healthService: "game.v1.SnapshotService",
			register: func(server *grpc.Server) {
				statev1.RegisterSnapshotServiceServer(server, snapshotService)
			},
		},
		{
			healthService: "game.v1.SessionService",
			register: func(server *grpc.Server) {
				statev1.RegisterSessionServiceServer(server, sessionService)
			},
		},
		{
			healthService: "game.v1.SceneService",
			register: func(server *grpc.Server) {
				statev1.RegisterSceneServiceServer(server, sceneService)
			},
		},
		{
			healthService: "game.v1.ForkService",
			register: func(server *grpc.Server) {
				statev1.RegisterForkServiceServer(server, forkService)
			},
		},
		{
			healthService: "game.v1.EventService",
			register: func(server *grpc.Server) {
				statev1.RegisterEventServiceServer(server, eventService)
			},
		},
		{
			healthService: "game.v1.StatisticsService",
			register: func(server *grpc.Server) {
				statev1.RegisterStatisticsServiceServer(server, statisticsService)
			},
		},
		{
			healthService: "game.v1.SystemService",
			register: func(server *grpc.Server) {
				statev1.RegisterSystemServiceServer(server, systemService)
			},
		},
		{
			healthService: "game.v1.AuthorizationService",
			register: func(server *grpc.Server) {
				statev1.RegisterAuthorizationServiceServer(server, authorizationService)
			},
		},
	}

	for _, descriptor := range descriptors {
		descriptor.register(grpcServer)
	}

	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	for _, descriptor := range descriptors {
		healthServer.SetServingStatus(descriptor.healthService, grpc_health_v1.HealthCheckResponse_SERVING)
	}
	return nil
}

func parseInternalServiceAllowlist(raw string) map[string]struct{} {
	values := strings.Split(strings.TrimSpace(raw), ",")
	allowlist := make(map[string]struct{}, len(values))
	for _, value := range values {
		serviceID := strings.ToLower(strings.TrimSpace(value))
		if serviceID == "" {
			continue
		}
		allowlist[serviceID] = struct{}{}
	}
	return allowlist
}
