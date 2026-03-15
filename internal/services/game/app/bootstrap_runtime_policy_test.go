package server

import (
	"context"
	"errors"
	"testing"

	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type projectionRuntimeConfigurerFunc func(serverEnv, *gamegrpc.Stores, storage.ProjectionApplyExactlyOnceStore, engine.Registries, *bridge.AdapterRegistry) (projectionRuntimeState, error)

func (f projectionRuntimeConfigurerFunc) Configure(
	srvEnv serverEnv,
	stores *gamegrpc.Stores,
	projectionStore storage.ProjectionApplyExactlyOnceStore,
	registries engine.Registries,
	adapters *bridge.AdapterRegistry,
) (projectionRuntimeState, error) {
	return f(srvEnv, stores, projectionStore, registries, adapters)
}

func TestConfigureProjectionRuntime_ConfiguresRuntimeAndOutboxBuilder(t *testing.T) {
	projectionRegistries := event.NewRegistry()
	if err := projectionRegistries.Register(event.Definition{
		Type:   event.Type("core.test_event"),
		Owner:  event.OwnerCore,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register projection event: %v", err)
	}

	var capturedStore storage.ProjectionApplyExactlyOnceStore
	var capturedRegistry *event.Registry
	applyCalled := false
	applyFn := func(context.Context, event.Event) error {
		applyCalled = true
		return nil
	}

	bootstrap := newServerBootstrapWithConfig(serverBootstrapConfig{
		projectionRuntimeConfigurer: projectionRuntimeConfigurerFunc(func(_ serverEnv, stores *gamegrpc.Stores, store storage.ProjectionApplyExactlyOnceStore, _ engine.Registries, _ *bridge.AdapterRegistry) (projectionRuntimeState, error) {
			if stores != nil && stores.Write.Runtime != nil {
				stores.Write.Runtime.SetInlineApplyEnabled(false)
				stores.Write.Runtime.SetIntentFilter(projectionRegistries)
			}
			capturedStore = store
			capturedRegistry = projectionRegistries
			return projectionRuntimeState{
				enableApplyWorker: true,
				applyOutbox:       applyFn,
			}, nil
		}),
	})

	var stores gamegrpc.Stores
	stores.Write.Runtime = gamegrpc.NewWriteRuntime()
	state, err := bootstrap.config.projectionRuntimeConfigurer.Configure(serverEnv{}, &stores, nil, engine.Registries{}, nil)
	if err != nil {
		t.Fatalf("configure projection runtime: %v", err)
	}

	if !state.enableApplyWorker {
		t.Fatal("expected apply worker to be enabled")
	}
	if state.enableShadowWorker {
		t.Fatal("expected shadow worker to be disabled")
	}
	if state.applyOutbox == nil {
		t.Fatal("expected outbox apply callback")
	}
	if capturedStore != nil {
		t.Fatal("expected nil projection store to flow to outbox apply builder")
	}
	if capturedRegistry != projectionRegistries {
		t.Fatal("expected projection registry built for runtime to flow to outbox apply builder")
	}

	if stores.Write.Runtime.InlineApplyEnabled() {
		t.Fatal("expected inline apply to be disabled in outbox-apply mode")
	}
	if !stores.Write.Runtime.ShouldApply()(event.Event{Type: event.Type("core.test_event")}) {
		t.Fatal("expected runtime intent filter to allow registered projection event")
	}
	if stores.Write.Runtime.ShouldApply()(event.Event{Type: event.Type("core.unknown_event")}) {
		t.Fatal("expected runtime intent filter to fail closed for unknown event")
	}

	if err := state.applyOutbox(context.Background(), event.Event{}); err != nil {
		t.Fatalf("invoke outbox apply callback: %v", err)
	}
	if !applyCalled {
		t.Fatal("expected outbox apply callback to be invoked")
	}
}

func TestConfigureProjectionRuntime_ReturnsResolveModeError(t *testing.T) {
	wantErr := errors.New("invalid projection mode")
	bootstrap := newServerBootstrapWithConfig(serverBootstrapConfig{
		projectionRuntimeConfigurer: defaultProjectionRuntimeConfigurer{
			resolveProjectionApplyModes: func(serverEnv) (bool, bool, string, error) {
				return false, false, "", wantErr
			},
			buildProjectionRegistries: func(engine.Registries, *bridge.AdapterRegistry) (*event.Registry, error) {
				t.Fatal("expected projection registry builder not to run after mode-resolution failure")
				return nil, nil
			},
			buildProjectionApplyOutboxApply: func(storage.ProjectionApplyExactlyOnceStore, *event.Registry) (func(context.Context, event.Event) error, error) {
				t.Fatal("expected outbox apply builder not to run after mode-resolution failure")
				return nil, nil
			},
		},
	})

	var stores gamegrpc.Stores
	stores.Write.Runtime = gamegrpc.NewWriteRuntime()
	_, err := bootstrap.config.projectionRuntimeConfigurer.Configure(serverEnv{}, &stores, nil, engine.Registries{}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected resolve-mode error %v, got %v", wantErr, err)
	}
}

func TestConfigureProjectionRuntime_ReturnsBuildProjectionRegistriesError(t *testing.T) {
	wantErr := errors.New("projection registry build failed")
	calledBuildApply := false
	bootstrap := newServerBootstrapWithConfig(serverBootstrapConfig{
		projectionRuntimeConfigurer: defaultProjectionRuntimeConfigurer{
			resolveProjectionApplyModes: func(serverEnv) (bool, bool, string, error) {
				return false, false, projectionApplyModeInlineApplyOnly, nil
			},
			buildProjectionRegistries: func(engine.Registries, *bridge.AdapterRegistry) (*event.Registry, error) {
				return nil, wantErr
			},
			buildProjectionApplyOutboxApply: func(storage.ProjectionApplyExactlyOnceStore, *event.Registry) (func(context.Context, event.Event) error, error) {
				calledBuildApply = true
				return nil, nil
			},
		},
	})

	var stores gamegrpc.Stores
	stores.Write.Runtime = gamegrpc.NewWriteRuntime()
	stores.Write.Runtime.SetInlineApplyEnabled(false)

	_, err := bootstrap.config.projectionRuntimeConfigurer.Configure(serverEnv{}, &stores, nil, engine.Registries{}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected registry-build error %v, got %v", wantErr, err)
	}
	if !stores.Write.Runtime.InlineApplyEnabled() {
		t.Fatal("expected inline apply to be enabled in inline mode before registry-build failure")
	}
	if calledBuildApply {
		t.Fatal("expected outbox apply builder not to run after registry-build failure")
	}
}

func TestStatusRuntimeState_CatalogReadinessDegradesWithNilBundle(t *testing.T) {
	catalogState := evaluateCatalogCapabilityState(context.Background(), nil)
	if catalogState.Ready {
		t.Fatal("expected catalog readiness to be false when store is nil")
	}
}
