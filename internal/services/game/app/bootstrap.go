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
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
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
	newManagedConn                  func(context.Context, platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error)
	newGRPCServer                   func(*storageBundle, serverEnv) *grpc.Server
	newHealthServer                 func() *health.Server
	resolveProjectionApplyModes     func(serverEnv) (bool, bool, string, error)
	buildProjectionRegistries       func(engine.Registries, *bridge.AdapterRegistry) (*event.Registry, error)
	buildProjectionApplyOutboxApply func(projectionApplyStore, *event.Registry) (func(context.Context, event.Event) error, error)
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
	if cfg.newManagedConn == nil {
		cfg.newManagedConn = platformgrpc.NewManagedConn
	}
	if cfg.newGRPCServer == nil {
		cfg.newGRPCServer = func(bundle *storageBundle, srvEnv serverEnv) *grpc.Server {
			internalIdentity := interceptors.InternalServiceIdentityConfig{
				MethodPrefixes: []string{
					"/game.v1.CampaignAIService/",
					campaignv1.EventService_AppendEvent_FullMethodName,
				},
				AllowedServiceIDs: parseInternalServiceAllowlist(srvEnv.InternalServiceAllowlist),
			}
			return grpc.NewServer(
				grpc.StatsHandler(otelgrpc.NewServerHandler()),
				grpc.ChainUnaryInterceptor(
					grpcmeta.UnaryServerInterceptor(nil),
					interceptors.InternalServiceIdentityUnaryInterceptor(internalIdentity),
					interceptors.AuditInterceptor(bundle.events),
					interceptors.SessionLockInterceptor(bundle.projections),
					interceptors.ErrorConversionUnaryInterceptor(),
				),
				grpc.ChainStreamInterceptor(
					grpcmeta.StreamServerInterceptor(nil),
					interceptors.InternalServiceIdentityStreamInterceptor(internalIdentity),
					interceptors.StreamAuditInterceptor(bundle.events),
					interceptors.ErrorConversionStreamInterceptor(),
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

// dependencyConns holds all outbound ManagedConns created during bootstrap.
type dependencyConns struct {
	auth   *platformgrpc.ManagedConn
	social *platformgrpc.ManagedConn
	ai     *platformgrpc.ManagedConn
	status *platformgrpc.ManagedConn

	// statusBindDone is closed when the background goroutine that late-binds
	// the status reporter client exits. Callers should wait on this channel
	// during shutdown before closing statusMc or reporter.
	statusBindDone   <-chan struct{}
	statusBindCancel context.CancelFunc
}

func (b *serverBootstrap) dialDependencies(
	ctx context.Context,
	srvEnv serverEnv,
	reporter *platformstatus.Reporter,
) (dependencyConns, error) {
	logf := func(format string, args ...any) {
		log.Printf(format, args...)
	}
	newConn := b.config.newManagedConn

	authMc, err := newConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "auth",
		Addr: srvEnv.AuthAddr,
		Mode: platformgrpc.ModeRequired,
		Logf: logf,
	})
	if err != nil {
		return dependencyConns{}, fmt.Errorf("auth: %w", err)
	}

	socialMc, err := newConn(ctx, platformgrpc.ManagedConnConfig{
		Name:             "social",
		Addr:             srvEnv.SocialAddr,
		Mode:             platformgrpc.ModeOptional,
		Logf:             logf,
		StatusReporter:   reporter,
		StatusCapability: "game.social.integration",
	})
	if err != nil {
		authMc.Close()
		return dependencyConns{}, fmt.Errorf("social: %w", err)
	}

	aiMc, err := newConn(ctx, platformgrpc.ManagedConnConfig{
		Name:             "ai",
		Addr:             srvEnv.AIAddr,
		Mode:             platformgrpc.ModeOptional,
		Logf:             logf,
		StatusReporter:   reporter,
		StatusCapability: "game.ai.integration",
	})
	if err != nil {
		authMc.Close()
		socialMc.Close()
		return dependencyConns{}, fmt.Errorf("ai: %w", err)
	}

	// Status service — optional, late-binds reporter when ready.
	statusAddr := srvEnv.StatusAddr
	if strings.TrimSpace(statusAddr) == "" {
		statusAddr = serviceaddr.DefaultGRPCAddr(serviceaddr.ServiceStatus)
	}
	statusMc, err := newConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "status",
		Addr: statusAddr,
		Mode: platformgrpc.ModeOptional,
		Logf: logf,
	})
	if err != nil {
		authMc.Close()
		socialMc.Close()
		aiMc.Close()
		return dependencyConns{}, fmt.Errorf("status: %w", err)
	}

	// Late-bind the reporter's client when the status service becomes reachable.
	// The done channel is closed when the goroutine exits so the shutdown path
	// can wait for it before tearing down statusMc and reporter.
	bindCtx, bindCancel := context.WithCancel(ctx)
	bindDone := make(chan struct{})
	go func() {
		defer close(bindDone)
		if err := statusMc.WaitReady(bindCtx); err != nil {
			log.Printf("status reporter: failed to bind client: %v", err)
			return
		}
		reporter.SetClient(statusv1.NewStatusServiceClient(statusMc.Conn()))
	}()

	return dependencyConns{
		auth:             authMc,
		social:           socialMc,
		ai:               aiMc,
		status:           statusMc,
		statusBindDone:   bindDone,
		statusBindCancel: bindCancel,
	}, nil
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
	if stores != nil && stores.Write.Runtime != nil {
		stores.Write.Runtime.SetInlineApplyEnabled(projectionApplyMode != projectionApplyModeOutboxApplyOnly)
	}
	log.Printf("projection apply mode = %s", projectionApplyMode)

	projectionRegistries, err := b.config.buildProjectionRegistries(registries, adapters)
	if err != nil {
		return projectionRuntimeState{}, err
	}
	if stores != nil && stores.Write.Runtime != nil {
		stores.Write.Runtime.SetIntentFilter(projectionRegistries)
	}

	applyOutbox, err := b.config.buildProjectionApplyOutboxApply(projectionStore, projectionRegistries)
	if err != nil {
		return projectionRuntimeState{}, fmt.Errorf("build projection apply outbox: %w", err)
	}

	return projectionRuntimeState{
		enableApplyWorker:  enableApplyWorker,
		enableShadowWorker: enableShadowWorker,
		applyOutbox:        applyOutbox,
	}, nil
}

func nilCatalogReadinessStore(bundle *storageBundle) storage.DaggerheartCatalogReadinessStore {
	if bundle == nil {
		return nil
	}
	return bundle.content
}

// NewWithAddr builds a game server using named startup phases.
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

	// Status reporter — starts with nil client; bound later when statusMc is ready.
	reporter := platformstatus.NewReporter("game", nil)
	reporter.Register(capabilityGameCampaignService, platformstatus.Operational)

	catalogState := evaluateCatalogCapabilityState(startupCtx, nilCatalogReadinessStore(bundle))
	applyCatalogCapabilityState(reporter, catalogState)

	deps, err := b.dialDependencies(startupCtx, srvEnv, reporter)
	if err != nil {
		return nil, wrapStartupError(startupPhaseDependencies, "dial dependencies", err)
	}
	rollback.add(func() {
		closeManagedConn(deps.status, "status")
		closeManagedConn(deps.ai, "ai")
		closeManagedConn(deps.social, "social")
		closeManagedConn(deps.auth, "auth")
	})

	// Build gRPC clients from managed connections — conn is always non-nil.
	stores.Social = socialv1.NewSocialServiceClient(deps.social.Conn())

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
		authv1.NewAuthServiceClient(deps.auth.Conn()),
		aiv1.NewAgentServiceClient(deps.ai.Conn()),
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

	server = &Server{
		listener:                                 listener,
		grpcServer:                               grpcServer,
		health:                                   healthServer,
		stores:                                   bundle,
		authMc:                                   deps.auth,
		socialMc:                                 deps.social,
		aiMc:                                     deps.ai,
		statusMc:                                 deps.status,
		statusBindDone:                           deps.statusBindDone,
		statusBindCancel:                         deps.statusBindCancel,
		projectionApplyOutboxWorkerEnabled:       projectionRuntime.enableApplyWorker,
		projectionApplyOutboxApply:               projectionRuntime.applyOutbox,
		projectionApplyOutboxShadowWorkerEnabled: projectionRuntime.enableShadowWorker,
		statusReporter:                           reporter,
		catalogReadyAtStartup:                    catalogState.Ready,
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
