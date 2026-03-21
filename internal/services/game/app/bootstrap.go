package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
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
	loadEnv                     func() (serverEnv, error)
	listen                      func(network, address string) (net.Listener, error)
	openStorageBundle           storageBundleOpener
	configureDomain             func(serverEnv, *gamegrpc.InfrastructureStores, *gamegrpc.RuntimeStores, engine.Registries) error
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
	Configure(serverEnv, *gamegrpc.RuntimeStores, projection.ExactlyOnceStore, engine.Registries, *bridge.AdapterRegistry) (projectionRuntimeState, error)
}

type defaultProjectionRuntimeConfigurer struct {
	resolveProjectionApplyModes     func(serverEnv) (bool, bool, string, error)
	buildProjectionRegistries       func(engine.Registries, *bridge.AdapterRegistry) (*event.Registry, error)
	buildProjectionApplyOutboxApply func(projection.ExactlyOnceStore, *event.Registry) (func(context.Context, event.Event) error, error)
}

func (c defaultProjectionRuntimeConfigurer) Configure(
	srvEnv serverEnv,
	runtimeStores *gamegrpc.RuntimeStores,
	projectionStore projection.ExactlyOnceStore,
	registries engine.Registries,
	adapters *bridge.AdapterRegistry,
) (projectionRuntimeState, error) {
	enableApplyWorker, enableShadowWorker, projectionApplyMode, err := c.resolveProjectionApplyModes(srvEnv)
	if err != nil {
		return projectionRuntimeState{}, err
	}
	if runtimeStores != nil && runtimeStores.Write.Runtime != nil {
		runtimeStores.Write.Runtime.SetInlineApplyEnabled(projectionApplyMode != projectionApplyModeOutboxApplyOnly)
	}
	slog.Info("projection apply mode resolved", "mode", projectionApplyMode)

	projectionRegistries, err := c.buildProjectionRegistries(registries, adapters)
	if err != nil {
		return projectionRuntimeState{}, err
	}
	if runtimeStores != nil && runtimeStores.Write.Runtime != nil {
		runtimeStores.Write.Runtime.SetIntentFilter(projectionRegistries)
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
			buildProjectionApplyOutboxApply: projection.BuildExactlyOnceApply,
		}
	}
	return cfg
}

func startupContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return ctx, func() {}
}

type configuredDomainState struct {
	projectionStores     gamegrpc.ProjectionStores
	systemStores         gamegrpc.SystemStores
	infrastructureStores gamegrpc.InfrastructureStores
	contentStores        gamegrpc.ContentStores
	runtimeStores        gamegrpc.RuntimeStores
	applier              projection.Applier
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
) (configuredDomainState, error) {
	writeRuntime := gamegrpc.NewWriteRuntime()
	storeGroups := buildStoreGroupsFromSources(storesConstructionSources{
		projectionStore: bundle.projections,
		systemStores:    bundle.systemStores,
		eventStore:      bundle.events,
		auditStore:      bundle.events,
		contentStore:    bundle.content,
		runtimeConfig: gamegrpc.StoresRuntimeConfig{
			WriteRuntime: writeRuntime,
		},
	})
	if err := b.config.configureDomain(srvEnv, &storeGroups.infrastructure, &storeGroups.runtime, registries); err != nil {
		return configuredDomainState{}, err
	}
	if err := gamegrpc.ValidateRootStoreGroups(
		storeGroups.projection,
		storeGroups.system,
		storeGroups.infrastructure,
		storeGroups.content,
		storeGroups.runtime,
	); err != nil {
		return configuredDomainState{}, err
	}
	applier, err := buildApplierFromSources(applierConstructionSources{
		projectionStore: bundle.projections,
		systemStores:    storeGroups.system,
		auditStore:      storeGroups.infrastructure.Audit,
		events:          registries.Events,
	})
	if err != nil {
		return configuredDomainState{}, err
	}
	if err := applier.ValidateStorePreconditions(); err != nil {
		return configuredDomainState{}, err
	}
	return configuredDomainState{
		projectionStores:     storeGroups.projection,
		systemStores:         storeGroups.system,
		infrastructureStores: storeGroups.infrastructure,
		contentStores:        storeGroups.content,
		runtimeStores:        storeGroups.runtime,
		applier:              applier,
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
	srvEnv, err := b.loadEnvPhase()
	if err != nil {
		return nil, err
	}
	startupCtx, startupCancel := startupContext(ctx, srvEnv.StartupTimeout)
	defer startupCancel()
	rollback := startupRollback{}
	defer func() {
		if err != nil {
			rollback.cleanup()
		}
	}()

	registries, err := b.buildRegistriesPhase()
	if err != nil {
		return nil, err
	}

	listener, err := b.openListenerPhase(addr, &rollback)
	if err != nil {
		return nil, err
	}

	bundle, err := b.openStoragePhase(startupCtx, srvEnv, registries.Events, &rollback)
	if err != nil {
		return nil, err
	}

	domainState, err := b.configureStoresPhase(startupCtx, srvEnv, bundle, registries)
	if err != nil {
		return nil, err
	}

	systemState, err := b.bootstrapSystemsPhase(startupCtx, bundle, registries, domainState.applier)
	if err != nil {
		return nil, err
	}

	reporter, catalogState := b.prepareStatusPhase(startupCtx, bundle)

	deps, err := b.dialDependenciesPhase(startupCtx, srvEnv, reporter, &rollback)
	if err != nil {
		return nil, err
	}
	attachDependencyClients(&domainState.contentStores, deps)
	registration := buildRegistrationAssemblies(registrationAssemblySources{
		bundle:          bundle,
		projectionStore: bundle.projections,
		contentStore:    bundle.content,
		eventStore:      bundle.events,
		domainState:     domainState,
		authClient:      newAuthServiceClient(deps.auth.Conn()),
		aiAgentClient:   newAIAgentServiceClient(deps.ai.Conn()),
		systemRegistry:  systemState.systemRegistry,
	})

	transportState, err := b.bootstrapTransportPhase(
		bundle,
		srvEnv,
		registration.daggerheart,
		registration.campaign,
		registration.session,
		registration.infrastructure,
	)
	if err != nil {
		return nil, err
	}

	projectionRuntime, err := b.configureProjectionRuntimePhase(
		srvEnv,
		&domainState.runtimeStores,
		bundle.projections,
		registries,
		domainState.applier.Adapters,
	)
	if err != nil {
		return nil, err
	}
	server = buildServerPhase(listener, bundle, deps, transportState, projectionRuntime, reporter, catalogState)
	rollback.release()
	return server, nil
}
