package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/interceptors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

// serverBootstrap configures each startup phase for the game server.
//
// Startup phases (see docs/running/game-startup-phases.md):
//  1. Registries — command/event/system registries from game modules
//  2. Network — open gRPC listener
//  3. Storage — open event journal, projections, and content databases
//  4. Domain — wire stores, build projection applier
//  5. Systems — validate system parity, repair projection gaps
//  6. Dependencies — connect auth (required), social/AI/status (graceful)
//  7. Transport — register gRPC services and health checks
//  8. Runtime — configure projection workers and status reporting
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
	buildProjectionApplyOutboxApply func(projectionApplyStore, *event.Registry) func(context.Context, event.Event) error
	buildStatusRuntime              func(context.Context, string, *storageBundle, bool, bool) statusRuntimeState
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
					interceptors.StreamAuditInterceptor(bundle.events),
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
	if cfg.buildStatusRuntime == nil {
		cfg.buildStatusRuntime = buildStatusRuntime
	}
	return cfg
}

func startupContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

type configuredStores struct {
	stores  gamegrpc.Stores
	applier projection.Applier
}

type projectionRuntimeState struct {
	enableApplyWorker  bool
	enableShadowWorker bool
	applyOutbox        func(context.Context, event.Event) error
}

type statusRuntimeState struct {
	conn                  *grpc.ClientConn
	reporter              *platformstatus.Reporter
	catalogReadyAtStartup bool
}

func (b *serverBootstrap) configureStoresAndApplier(
	ctx context.Context,
	srvEnv serverEnv,
	bundle *storageBundle,
	registries engine.Registries,
) (configuredStores, error) {
	writeRuntime := gamegrpc.NewWriteRuntime()
	stores := gamegrpc.NewStoresFromProjection(gamegrpc.StoresFromProjectionConfig{
		ProjectionStore: bundle.projections,
		SystemStores:    systemmanifest.ExtractProjectionStores(bundle.projections),
		EventStore:      bundle.events,
		ContentStore:    bundle.content,
		WriteRuntime:    writeRuntime,
		Events:          registries.Events,
	})
	if err := b.config.configureDomain(srvEnv, &stores, registries); err != nil {
		return configuredStores{}, err
	}
	if err := stores.Validate(); err != nil {
		return configuredStores{}, err
	}
	applier, err := stores.TryApplier()
	if err != nil {
		return configuredStores{}, err
	}
	return configuredStores{
		stores:  stores,
		applier: applier,
	}, nil
}

func (b *serverBootstrap) dialDependencyClients(
	ctx context.Context,
	srvEnv serverEnv,
) (authGRPCClients, socialGRPCClients, aiGRPCClients, error) {
	authClients, err := b.config.dialAuthGRPC(ctx, srvEnv.AuthAddr)
	if err != nil {
		return authGRPCClients{}, socialGRPCClients{}, aiGRPCClients{}, err
	}

	socialClients, socialErr := b.config.dialSocialGRPC(ctx, srvEnv.SocialAddr)
	if socialErr != nil {
		log.Printf("social client unavailable; participant pronouns fallback disabled: %v", socialErr)
		socialClients = socialGRPCClients{}
	}

	aiClients, aiErr := b.config.dialAIGRPC(ctx, srvEnv.AIAddr)
	if aiErr != nil {
		log.Printf("ai client unavailable; campaign ai binding operations will be unavailable: %v", aiErr)
		aiClients = aiGRPCClients{}
	}

	return authClients, socialClients, aiClients, nil
}

func (b *serverBootstrap) configureProjectionRuntime(
	srvEnv serverEnv,
	stores *gamegrpc.Stores,
	projectionStore projectionApplyStore,
	registries engine.Registries,
	adapters *bridge.AdapterRegistry,
) (projectionRuntimeState, error) {
	enableApplyWorker, enableShadowWorker, projectionApplyMode, err := b.config.resolveProjectionApplyModes(srvEnv)
	if err != nil {
		return projectionRuntimeState{}, err
	}
	if stores != nil && stores.WriteRuntime != nil {
		stores.WriteRuntime.SetInlineApplyEnabled(projectionApplyMode != projectionApplyModeOutboxApplyOnly)
	}
	log.Printf("projection apply mode = %s", projectionApplyMode)

	projectionRegistries, err := b.config.buildProjectionRegistries(registries, adapters)
	if err != nil {
		return projectionRuntimeState{}, err
	}
	if stores != nil && stores.WriteRuntime != nil {
		stores.WriteRuntime.SetIntentFilter(projectionRegistries)
	}

	return projectionRuntimeState{
		enableApplyWorker:  enableApplyWorker,
		enableShadowWorker: enableShadowWorker,
		applyOutbox:        b.config.buildProjectionApplyOutboxApply(projectionStore, projectionRegistries),
	}, nil
}

func buildStatusRuntime(
	ctx context.Context,
	statusAddr string,
	bundle *storageBundle,
	socialAvailable, aiAvailable bool,
) statusRuntimeState {
	statusConn, statusClient := dialStatusLenient(ctx, statusAddr)
	catalogState := evaluateCatalogCapabilityState(ctx, nilCatalogReadinessStore(bundle))
	return statusRuntimeState{
		conn:                  statusConn,
		reporter:              initStatusReporter(statusClient, socialAvailable, aiAvailable, catalogState),
		catalogReadyAtStartup: catalogState.Ready,
	}
}

func nilCatalogReadinessStore(bundle *storageBundle) storage.DaggerheartCatalogReadinessStore {
	if bundle == nil {
		return nil
	}
	return bundle.content
}

func (b *serverBootstrap) configureStatusRuntime(
	ctx context.Context,
	statusAddr string,
	bundle *storageBundle,
	socialAvailable, aiAvailable bool,
) statusRuntimeState {
	return b.config.buildStatusRuntime(ctx, statusAddr, bundle, socialAvailable, aiAvailable)
}

// NewWithAddr builds a game server using named startup phases.
// It preserves the previous startup behavior while improving testability through dependency seams.
func (b *serverBootstrap) NewWithAddr(ctx context.Context, addr string) (server *Server, err error) {
	startupCtx := startupContext(ctx)
	srvEnv := b.config.loadEnv()
	rollback := startupRollback{}
	defer func() {
		if err != nil {
			rollback.cleanup()
		}
	}()

	registries, err := engine.BuildRegistries(registeredSystemModules()...)
	if err != nil {
		return nil, wrapStartupError(startupPhaseRegistries, "build registries", err)
	}

	listener, err := b.config.listen("tcp", addr)
	if err != nil {
		return nil, wrapStartupError(startupPhaseNetwork, fmt.Sprintf("listen on %s", addr), err)
	}
	rollback.add(func() {
		_ = listener.Close()
	})

	bundle, err := b.config.openStorageBundle.Open(startupCtx, srvEnv, registries.Events)
	if err != nil {
		return nil, wrapStartupError(startupPhaseStorage, "open storage bundle", err)
	}
	rollback.add(func() {
		bundle.Close()
	})

	storeState, err := b.configureStoresAndApplier(startupCtx, srvEnv, bundle, registries)
	if err != nil {
		return nil, wrapStartupError(startupPhaseDomain, "configure stores and applier", err)
	}
	stores := storeState.stores

	systemRegistry, err := b.config.buildSystemRegistry()
	if err != nil {
		return nil, wrapStartupError(startupPhaseSystems, "build system registry", err)
	}

	if err := b.config.validateSystemRegistration(registeredSystemModules(), systemRegistry, storeState.applier.Adapters); err != nil {
		return nil, wrapStartupError(startupPhaseSystems, "validate system parity", err)
	}
	if err := interceptors.ValidateSessionLockPolicyCoverage(interceptors.BlockedCommandNamespaces()); err != nil {
		return nil, wrapStartupError(startupPhaseSystems, "validate session lock policy", err)
	}
	repairProjectionGaps(startupCtx, bundle, storeState.applier)

	authClients, socialClients, aiClients, err := b.dialDependencyClients(startupCtx, srvEnv)
	if err != nil {
		return nil, wrapStartupError(startupPhaseDependencies, "dial auth gRPC", err)
	}
	if authClients.conn != nil {
		authConn := authClients.conn
		rollback.add(func() {
			_ = authConn.Close()
		})
	}
	if socialClients.conn != nil {
		socialConn := socialClients.conn
		rollback.add(func() {
			_ = socialConn.Close()
		})
	}
	if aiClients.conn != nil {
		aiConn := aiClients.conn
		rollback.add(func() {
			_ = aiConn.Close()
		})
	}
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

	projectionRuntime, err := b.configureProjectionRuntime(
		srvEnv,
		&stores,
		bundle.projections,
		registries,
		storeState.applier.Adapters,
	)
	if err != nil {
		return nil, wrapStartupError(startupPhaseRuntime, "configure projection runtime", err)
	}

	statusRuntime := b.configureStatusRuntime(
		startupCtx,
		srvEnv.StatusAddr,
		bundle,
		socialClients.socialClient != nil,
		aiClients.agentClient != nil,
	)
	if statusRuntime.conn != nil {
		statusConn := statusRuntime.conn
		rollback.add(func() {
			_ = statusConn.Close()
		})
	}

	server = &Server{
		listener:                                 listener,
		grpcServer:                               grpcServer,
		health:                                   healthServer,
		stores:                                   bundle,
		authConn:                                 authClients.conn,
		socialConn:                               socialClients.conn,
		aiConn:                                   aiClients.conn,
		statusConn:                               statusRuntime.conn,
		projectionApplyOutboxWorkerEnabled:       projectionRuntime.enableApplyWorker,
		projectionApplyOutboxApply:               projectionRuntime.applyOutbox,
		projectionApplyOutboxShadowWorkerEnabled: projectionRuntime.enableShadowWorker,
		statusReporter:                           statusRuntime.reporter,
		catalogReadyAtStartup:                    statusRuntime.catalogReadyAtStartup,
	}
	rollback.release()
	return server, nil
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
