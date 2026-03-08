package server

import (
	"context"
	"errors"
	"testing"

	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestConfigureProjectionRuntime_ConfiguresRuntimeAndOutboxBuilder(t *testing.T) {
	projectionRegistries := event.NewRegistry()
	if err := projectionRegistries.Register(event.Definition{
		Type:   event.Type("core.test_event"),
		Owner:  event.OwnerCore,
		Intent: event.IntentProjectionAndReplay,
	}); err != nil {
		t.Fatalf("register projection event: %v", err)
	}

	var capturedStore projectionApplyStore
	var capturedRegistry *event.Registry
	applyCalled := false
	applyFn := func(context.Context, event.Event) error {
		applyCalled = true
		return nil
	}

	bootstrap := newServerBootstrapWithConfig(serverBootstrapConfig{
		resolveProjectionApplyModes: func(serverEnv) (bool, bool, string, error) {
			return true, false, projectionApplyModeOutboxApplyOnly, nil
		},
		buildProjectionRegistries: func(engine.Registries, *bridge.AdapterRegistry) (*event.Registry, error) {
			return projectionRegistries, nil
		},
		buildProjectionApplyOutboxApply: func(store projectionApplyStore, registries *event.Registry) func(context.Context, event.Event) error {
			capturedStore = store
			capturedRegistry = registries
			return applyFn
		},
	})

	var stores gamegrpc.Stores
	stores.Write.Runtime = gamegrpc.NewWriteRuntime()
	state, err := bootstrap.configureProjectionRuntime(serverEnv{}, &stores, nil, engine.Registries{}, nil)
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
	calledBuildRegistries := false
	calledBuildApply := false
	bootstrap := newServerBootstrapWithConfig(serverBootstrapConfig{
		resolveProjectionApplyModes: func(serverEnv) (bool, bool, string, error) {
			return false, false, "", wantErr
		},
		buildProjectionRegistries: func(engine.Registries, *bridge.AdapterRegistry) (*event.Registry, error) {
			calledBuildRegistries = true
			return nil, nil
		},
		buildProjectionApplyOutboxApply: func(projectionApplyStore, *event.Registry) func(context.Context, event.Event) error {
			calledBuildApply = true
			return nil
		},
	})

	var stores gamegrpc.Stores
	stores.Write.Runtime = gamegrpc.NewWriteRuntime()
	_, err := bootstrap.configureProjectionRuntime(serverEnv{}, &stores, nil, engine.Registries{}, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected resolve-mode error %v, got %v", wantErr, err)
	}
	if calledBuildRegistries {
		t.Fatal("expected projection registry builder not to run after mode-resolution failure")
	}
	if calledBuildApply {
		t.Fatal("expected outbox apply builder not to run after mode-resolution failure")
	}
}

func TestConfigureProjectionRuntime_ReturnsBuildProjectionRegistriesError(t *testing.T) {
	wantErr := errors.New("projection registry build failed")
	calledBuildApply := false
	bootstrap := newServerBootstrapWithConfig(serverBootstrapConfig{
		resolveProjectionApplyModes: func(serverEnv) (bool, bool, string, error) {
			return false, false, projectionApplyModeInlineApplyOnly, nil
		},
		buildProjectionRegistries: func(engine.Registries, *bridge.AdapterRegistry) (*event.Registry, error) {
			return nil, wantErr
		},
		buildProjectionApplyOutboxApply: func(projectionApplyStore, *event.Registry) func(context.Context, event.Event) error {
			calledBuildApply = true
			return nil
		},
	})

	var stores gamegrpc.Stores
	stores.Write.Runtime = gamegrpc.NewWriteRuntime()
	stores.Write.Runtime.SetInlineApplyEnabled(false)

	_, err := bootstrap.configureProjectionRuntime(serverEnv{}, &stores, nil, engine.Registries{}, nil)
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

func TestBuildStatusRuntime_NilBundleReturnsReporterAndDegradedCatalog(t *testing.T) {
	state := buildStatusRuntime(context.Background(), "", nil, false, false)
	if state.conn != nil {
		t.Fatal("expected nil status connection when status address is empty")
	}
	if state.reporter == nil {
		t.Fatal("expected status reporter to be initialized")
	}
	if state.catalogReadyAtStartup {
		t.Fatal("expected catalog readiness to be false when content store is unavailable")
	}
}

func TestConfigureStatusRuntime_UsesConfiguredBuilder(t *testing.T) {
	called := false
	var gotAddr string
	var gotBundle *storageBundle
	var gotSocial bool
	var gotAI bool
	wantState := statusRuntimeState{
		catalogReadyAtStartup: true,
	}
	bootstrap := newServerBootstrapWithConfig(serverBootstrapConfig{
		buildStatusRuntime: func(_ context.Context, statusAddr string, bundle *storageBundle, socialAvailable, aiAvailable bool) statusRuntimeState {
			called = true
			gotAddr = statusAddr
			gotBundle = bundle
			gotSocial = socialAvailable
			gotAI = aiAvailable
			return wantState
		},
	})

	bundle := &storageBundle{}
	state := bootstrap.configureStatusRuntime(context.Background(), "status:9000", bundle, true, false)
	if !called {
		t.Fatal("expected configured status runtime builder to be called")
	}
	if gotAddr != "status:9000" {
		t.Fatalf("expected status address status:9000, got %s", gotAddr)
	}
	if gotBundle != bundle {
		t.Fatal("expected storage bundle to be forwarded to status runtime builder")
	}
	if !gotSocial {
		t.Fatal("expected social availability flag to be forwarded")
	}
	if gotAI {
		t.Fatal("expected ai availability flag to be forwarded as false")
	}
	if state.catalogReadyAtStartup != wantState.catalogReadyAtStartup {
		t.Fatal("expected status runtime state from configured builder")
	}
}
