package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
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
	loadEnv                     func() serverEnv
	listen                      func(network, address string) (net.Listener, error)
	openStorageBundle           storageBundleOpener
	configureDomain             func(serverEnv, *gamegrpc.Stores, engine.Registries) error
	systemsBootstrapper         systemsBootstrapper
	dependencyDialer            dependencyDialer
	transportBootstrapper       transportBootstrapper
	projectionRuntimeConfigurer projectionRuntimeConfigurer
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

// dependencyDialer owns startup-time outbound service connection wiring.
type dependencyDialer interface {
	Dial(context.Context, serverEnv, *platformstatus.Reporter) (dependencyConns, error)
}

type managedConnDependencyDialer struct {
	newManagedConn func(context.Context, platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error)
}

func (d managedConnDependencyDialer) Dial(
	ctx context.Context,
	srvEnv serverEnv,
	reporter *platformstatus.Reporter,
) (dependencyConns, error) {
	authMc, err := d.newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "auth",
		Addr: srvEnv.AuthAddr,
		Mode: platformgrpc.ModeRequired,
		Logf: startupLogf,
	})
	if err != nil {
		return dependencyConns{}, fmt.Errorf("auth: %w", err)
	}

	socialMc, err := d.newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name:             "social",
		Addr:             srvEnv.SocialAddr,
		Mode:             platformgrpc.ModeOptional,
		Logf:             startupLogf,
		StatusReporter:   reporter,
		StatusCapability: "game.social.integration",
	})
	if err != nil {
		authMc.Close()
		return dependencyConns{}, fmt.Errorf("social: %w", err)
	}

	aiMc, err := d.newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name:             "ai",
		Addr:             srvEnv.AIAddr,
		Mode:             platformgrpc.ModeOptional,
		Logf:             startupLogf,
		StatusReporter:   reporter,
		StatusCapability: "game.ai.integration",
	})
	if err != nil {
		authMc.Close()
		socialMc.Close()
		return dependencyConns{}, fmt.Errorf("ai: %w", err)
	}

	statusAddr := srvEnv.StatusAddr
	if strings.TrimSpace(statusAddr) == "" {
		statusAddr = serviceaddr.DefaultGRPCAddr(serviceaddr.ServiceStatus)
	}
	statusMc, err := d.newManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "status",
		Addr: statusAddr,
		Mode: platformgrpc.ModeOptional,
		Logf: startupLogf,
	})
	if err != nil {
		authMc.Close()
		socialMc.Close()
		aiMc.Close()
		return dependencyConns{}, fmt.Errorf("status: %w", err)
	}

	bindCtx, bindCancel := context.WithCancel(ctx)
	bindDone := make(chan struct{})
	go func() {
		defer close(bindDone)
		if err := statusMc.WaitReady(bindCtx); err != nil {
			startupLogf("status reporter: failed to bind client: %v", err)
			return
		}
		reporter.SetClient(newStatusServiceClient(statusMc.Conn()))
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

// projectionRuntimeConfigurer owns startup-time projection worker and
// inline-apply runtime configuration.
type projectionRuntimeConfigurer interface {
	Configure(serverEnv, *gamegrpc.Stores, projectionApplyStore, engine.Registries, *bridge.AdapterRegistry) (projectionRuntimeState, error)
}

type defaultProjectionRuntimeConfigurer struct {
	resolveProjectionApplyModes     func(serverEnv) (bool, bool, string, error)
	buildProjectionRegistries       func(engine.Registries, *bridge.AdapterRegistry) (*event.Registry, error)
	buildProjectionApplyOutboxApply func(projectionApplyStore, *event.Registry) (func(context.Context, event.Event) error, error)
}

func (c defaultProjectionRuntimeConfigurer) Configure(
	srvEnv serverEnv,
	stores *gamegrpc.Stores,
	projectionStore projectionApplyStore,
	registries engine.Registries,
	adapters *bridge.AdapterRegistry,
) (projectionRuntimeState, error) {
	enableApplyWorker, enableShadowWorker, projectionApplyMode, err := c.resolveProjectionApplyModes(srvEnv)
	if err != nil {
		return projectionRuntimeState{}, err
	}
	if stores != nil && stores.Write.Runtime != nil {
		stores.Write.Runtime.SetInlineApplyEnabled(projectionApplyMode != projectionApplyModeOutboxApplyOnly)
	}
	log.Printf("projection apply mode = %s", projectionApplyMode)

	projectionRegistries, err := c.buildProjectionRegistries(registries, adapters)
	if err != nil {
		return projectionRuntimeState{}, err
	}
	if stores != nil && stores.Write.Runtime != nil {
		stores.Write.Runtime.SetIntentFilter(projectionRegistries)
	}

	applyOutbox, err := c.buildProjectionApplyOutboxApply(projectionStore, projectionRegistries)
	if err != nil {
		return projectionRuntimeState{}, fmt.Errorf("build projection apply outbox: %w", err)
	}

	return projectionRuntimeState{
		enableApplyWorker:  enableApplyWorker,
		enableShadowWorker: enableShadowWorker,
		applyOutbox:        applyOutbox,
	}, nil
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
	if cfg.systemsBootstrapper == nil {
		cfg.systemsBootstrapper = defaultSystemsBootstrapper{
			buildSystemRegistry:        buildSystemRegistry,
			validateSystemRegistration: validateSystemRegistrationParity,
			validateSessionLockPolicy:  validateSessionLockPolicy,
			repairProjectionGaps:       repairProjectionGaps,
		}
	}
	if cfg.dependencyDialer == nil {
		cfg.dependencyDialer = managedConnDependencyDialer{newManagedConn: platformgrpc.NewManagedConn}
	}
	if cfg.transportBootstrapper == nil {
		cfg.transportBootstrapper = defaultTransportBootstrapper{
			newGRPCServer:            newDefaultGRPCServer,
			newHealthServer:          newDefaultHealthServer,
			loadAISessionGrantConfig: loadAISessionGrantConfig,
			registerServices:         transportServiceRegistrarFunc(registerServices),
		}
	}
	if cfg.projectionRuntimeConfigurer == nil {
		cfg.projectionRuntimeConfigurer = defaultProjectionRuntimeConfigurer{
			resolveProjectionApplyModes:     resolveProjectionApplyOutboxModes,
			buildProjectionRegistries:       buildProjectionRegistries,
			buildProjectionApplyOutboxApply: buildProjectionApplyOutboxApply,
		}
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
		SystemStores:    gamegrpc.SystemStores{Daggerheart: bundle.projections.DaggerheartProjectionStore()},
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

func nilCatalogReadinessStore(bundle *storageBundle) contentstore.DaggerheartCatalogReadinessStore {
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

	systemState, err := b.config.systemsBootstrapper.Bootstrap(startupCtx, bundle, registries, storeState.applier)
	if err != nil {
		return nil, wrapStartupError(startupPhaseSystems, "bootstrap systems phase", err)
	}

	// Status reporter — starts with nil client; bound later when statusMc is ready.
	reporter := platformstatus.NewReporter("game", nil)
	reporter.Register(capabilityGameCampaignService, platformstatus.Operational)

	catalogState := evaluateCatalogCapabilityState(startupCtx, nilCatalogReadinessStore(bundle))
	applyCatalogCapabilityState(reporter, catalogState)

	deps, err := b.config.dependencyDialer.Dial(startupCtx, srvEnv, reporter)
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

	transportState, err := b.config.transportBootstrapper.Bootstrap(
		bundle,
		srvEnv,
		stores,
		newAuthServiceClient(deps.auth.Conn()),
		newAIAgentServiceClient(deps.ai.Conn()),
		systemState.systemRegistry,
	)
	if err != nil {
		return nil, wrapStartupError(startupPhaseTransport, "bootstrap transport phase", err)
	}

	projectionRuntime, err := b.config.projectionRuntimeConfigurer.Configure(
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
		grpcServer:                               transportState.grpcServer,
		health:                                   transportState.healthServer,
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
