package app

import (
	"context"
	"fmt"
	"net"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

func (b *serverBootstrap) loadEnvPhase() (serverEnv, error) {
	srvEnv, err := b.config.loadEnv()
	if err != nil {
		return serverEnv{}, wrapStartupError(startupPhaseRegistries, "load server env", err)
	}
	return srvEnv, nil
}

func (b *serverBootstrap) buildRegistriesPhase() (engine.Registries, error) {
	registries, err := engine.BuildRegistries(registeredSystemModules()...)
	if err != nil {
		return engine.Registries{}, wrapStartupError(startupPhaseRegistries, "build registries", err)
	}
	return registries, nil
}

func (b *serverBootstrap) openListenerPhase(addr string, rollback *startupRollback) (net.Listener, error) {
	listener, err := b.config.listen("tcp", addr)
	if err != nil {
		return nil, wrapStartupError(startupPhaseNetwork, fmt.Sprintf("listen on %s", addr), err)
	}
	if rollback != nil {
		rollback.add(func() {
			_ = listener.Close()
		})
	}
	return listener, nil
}

func (b *serverBootstrap) openStoragePhase(
	ctx context.Context,
	srvEnv serverEnv,
	eventRegistry *event.Registry,
	rollback *startupRollback,
) (*storageBundle, error) {
	bundle, err := b.config.openStorageBundle.Open(ctx, srvEnv, eventRegistry)
	if err != nil {
		return nil, wrapStartupError(startupPhaseStorage, "open storage bundle", err)
	}
	if rollback != nil {
		rollback.add(func() {
			bundle.Close()
		})
	}
	return bundle, nil
}

func (b *serverBootstrap) configureStoresPhase(
	ctx context.Context,
	srvEnv serverEnv,
	bundle *storageBundle,
	registries engine.Registries,
) (configuredDomainState, error) {
	storeState, err := b.configureStoresAndApplier(ctx, srvEnv, bundle, registries)
	if err != nil {
		return configuredDomainState{}, wrapStartupError(startupPhaseDomain, "configure stores and applier", err)
	}
	return storeState, nil
}

func (b *serverBootstrap) bootstrapSystemsPhase(
	ctx context.Context,
	bundle *storageBundle,
	registries engine.Registries,
	applier projection.Applier,
) (systemsRuntimeState, error) {
	systemState, err := b.config.systemsBootstrapper.Bootstrap(ctx, bundle, registries, applier)
	if err != nil {
		return systemsRuntimeState{}, wrapStartupError(startupPhaseSystems, "bootstrap systems phase", err)
	}
	return systemState, nil
}

func (b *serverBootstrap) prepareStatusPhase(
	ctx context.Context,
	bundle *storageBundle,
) (*platformstatus.Reporter, catalogCapabilityState) {
	reporter := platformstatus.NewReporter("game", nil)
	reporter.Register(capabilityGameCampaignService, platformstatus.Operational)

	catalogState := evaluateCatalogCapabilityState(ctx, nilCatalogReadinessStore(bundle))
	applyCatalogCapabilityState(reporter, catalogState)
	return reporter, catalogState
}

func (b *serverBootstrap) dialDependenciesPhase(
	ctx context.Context,
	srvEnv serverEnv,
	reporter *platformstatus.Reporter,
	rollback *startupRollback,
) (dependencyConns, error) {
	deps, err := b.config.dependencyDialer.Dial(ctx, srvEnv, reporter)
	if err != nil {
		return dependencyConns{}, wrapStartupError(startupPhaseDependencies, "dial dependencies", err)
	}
	if rollback != nil {
		rollback.add(func() {
			closeManagedConn(deps.status, "status")
			closeManagedConn(deps.ai, "ai")
			closeManagedConn(deps.social, "social")
			closeManagedConn(deps.auth, "auth")
		})
	}
	return deps, nil
}

func attachDependencyClients(contentStores *gamegrpc.ContentStores, deps dependencyConns) {
	if contentStores == nil {
		return
	}
	contentStores.Social = socialv1.NewSocialServiceClient(deps.social.Conn())
}

func (b *serverBootstrap) bootstrapTransportPhase(
	bundle *storageBundle,
	srvEnv serverEnv,
	daggerheartDeps daggerheartRegistrationDeps,
	campaignDeps campaignRegistrationDeps,
	sessionDeps sessionRegistrationDeps,
	infrastructureDeps infrastructureRegistrationDeps,
) (transportRuntimeState, error) {
	transportState, err := b.config.transportBootstrapper.Bootstrap(
		bundle,
		srvEnv,
		daggerheartDeps,
		campaignDeps,
		sessionDeps,
		infrastructureDeps,
	)
	if err != nil {
		return transportRuntimeState{}, wrapStartupError(startupPhaseTransport, "bootstrap transport phase", err)
	}
	return transportState, nil
}

func (b *serverBootstrap) configureProjectionRuntimePhase(
	srvEnv serverEnv,
	runtimeStores *gamegrpc.RuntimeStores,
	projectionStore projection.ExactlyOnceStore,
	registries engine.Registries,
	adapters *bridge.AdapterRegistry,
) (projectionRuntimeState, error) {
	projectionRuntime, err := b.config.projectionRuntimeConfigurer.Configure(
		srvEnv,
		runtimeStores,
		projectionStore,
		registries,
		adapters,
	)
	if err != nil {
		return projectionRuntimeState{}, wrapStartupError(startupPhaseRuntime, "configure projection runtime", err)
	}
	return projectionRuntime, nil
}

func buildServerPhase(
	listener net.Listener,
	bundle *storageBundle,
	deps dependencyConns,
	transport transportRuntimeState,
	projectionRuntime projectionRuntimeState,
	reporter *platformstatus.Reporter,
	catalogState catalogCapabilityState,
) *Server {
	return &Server{
		listener: listener,
		stores:   bundle,
		transport: transportState{
			grpcServer: transport.grpcServer,
			health:     transport.healthServer,
		},
		conns: connectionState{
			authMc:           deps.auth,
			socialMc:         deps.social,
			aiMc:             deps.ai,
			statusMc:         deps.status,
			statusBindDone:   deps.statusBindDone,
			statusBindCancel: deps.statusBindCancel,
		},
		workers: projectionWorkerState{
			applyWorkerEnabled:  projectionRuntime.enableApplyWorker,
			applyFunc:           projectionRuntime.applyOutbox,
			shadowWorkerEnabled: projectionRuntime.enableShadowWorker,
		},
		status: statusState{
			reporter:              reporter,
			catalogReadyAtStartup: catalogState.Ready,
		},
	}
}
