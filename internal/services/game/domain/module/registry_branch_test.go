package module

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

type nilReadinessProviderModule struct {
	stubModule
}

func (m nilReadinessProviderModule) BindCharacterReadiness(ids.CampaignID, map[Key]any) (CharacterReadinessEvaluator, error) {
	return nil, nil
}

type nilBootstrapProviderModule struct {
	stubModule
}

func (m nilBootstrapProviderModule) BindSessionStartBootstrap(ids.CampaignID, map[Key]any) (SessionStartBootstrapEmitter, error) {
	return nil, nil
}

func TestResolveSnapshotState_AllowsModulesWithoutFactory(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	mod, state, err := ResolveSnapshotState(registry, "camp-1", "daggerheart", "v1", nil)
	if err != nil {
		t.Fatalf("ResolveSnapshotState() error = %v", err)
	}
	if mod == nil {
		t.Fatal("expected resolved module")
	}
	if state != nil {
		t.Fatalf("state = %v, want nil", state)
	}
}

func TestResolveSnapshotState_ReportsResolutionError(t *testing.T) {
	_, _, err := ResolveSnapshotState(nil, "camp-1", "daggerheart", "v1", nil)
	if err != ErrRegistryRequired {
		t.Fatalf("expected ErrRegistryRequired, got %v", err)
	}
}

func TestResolveCharacterReadiness_DisabledForMissingInputs(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		evaluator, enabled, err := ResolveCharacterReadiness(nil, "camp-1", "daggerheart", nil)
		if err != nil {
			t.Fatalf("ResolveCharacterReadiness() error = %v, want nil", err)
		}
		if enabled || evaluator != nil {
			t.Fatalf("result = (%v, %t), want (nil, false)", evaluator, enabled)
		}
	})

	t.Run("blank system id", func(t *testing.T) {
		registry := NewRegistry()
		evaluator, enabled, err := ResolveCharacterReadiness(registry, "camp-1", "  ", nil)
		if err != nil {
			t.Fatalf("ResolveCharacterReadiness() error = %v, want nil", err)
		}
		if enabled || evaluator != nil {
			t.Fatalf("result = (%v, %t), want (nil, false)", evaluator, enabled)
		}
	})

	t.Run("missing module", func(t *testing.T) {
		registry := NewRegistry()
		evaluator, enabled, err := ResolveCharacterReadiness(registry, "camp-1", "daggerheart", nil)
		if err != nil {
			t.Fatalf("ResolveCharacterReadiness() error = %v, want nil", err)
		}
		if enabled || evaluator != nil {
			t.Fatalf("result = (%v, %t), want (nil, false)", evaluator, enabled)
		}
	})
}

func TestResolveCharacterReadiness_RejectsNilEvaluator(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(nilReadinessProviderModule{
		stubModule: stubModule{id: "daggerheart", version: "v1"},
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	evaluator, enabled, err := ResolveCharacterReadiness(registry, "camp-1", "daggerheart", nil)
	if err == nil {
		t.Fatal("expected nil evaluator error")
	}
	if !enabled {
		t.Fatal("expected readiness hook to be enabled")
	}
	if evaluator != nil {
		t.Fatalf("evaluator = %v, want nil", evaluator)
	}
}

func TestResolveSessionStartBootstrap_DisabledForMissingInputs(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		emitter, enabled, err := ResolveSessionStartBootstrap(nil, "camp-1", "daggerheart", nil)
		if err != nil {
			t.Fatalf("ResolveSessionStartBootstrap() error = %v, want nil", err)
		}
		if enabled || emitter != nil {
			t.Fatalf("result = (%v, %t), want (nil, false)", emitter, enabled)
		}
	})

	t.Run("blank system id", func(t *testing.T) {
		registry := NewRegistry()
		emitter, enabled, err := ResolveSessionStartBootstrap(registry, "camp-1", "  ", nil)
		if err != nil {
			t.Fatalf("ResolveSessionStartBootstrap() error = %v, want nil", err)
		}
		if enabled || emitter != nil {
			t.Fatalf("result = (%v, %t), want (nil, false)", emitter, enabled)
		}
	})

	t.Run("missing module", func(t *testing.T) {
		registry := NewRegistry()
		emitter, enabled, err := ResolveSessionStartBootstrap(registry, "camp-1", "daggerheart", nil)
		if err != nil {
			t.Fatalf("ResolveSessionStartBootstrap() error = %v, want nil", err)
		}
		if enabled || emitter != nil {
			t.Fatalf("result = (%v, %t), want (nil, false)", emitter, enabled)
		}
	})
}

func TestResolveSessionStartBootstrap_RejectsNilEmitter(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(nilBootstrapProviderModule{
		stubModule: stubModule{id: "daggerheart", version: "v1"},
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	emitter, enabled, err := ResolveSessionStartBootstrap(registry, "camp-1", "daggerheart", nil)
	if err == nil {
		t.Fatal("expected nil emitter error")
	}
	if !enabled {
		t.Fatal("expected bootstrap hook to be enabled")
	}
	if emitter != nil {
		t.Fatalf("emitter = %v, want nil", emitter)
	}
}

func TestResolveSessionStartBootstrap_ReportsBindError(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(bootstrapHookModule{
		stubModule: stubModule{
			id:      "daggerheart",
			version: "v1",
			factory: stubFactory{snapshotErr: errSentinel("boom")},
		},
	}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	emitter, enabled, err := ResolveSessionStartBootstrap(registry, "camp-1", "daggerheart", nil)
	if err == nil {
		t.Fatal("expected bind error")
	}
	if !enabled {
		t.Fatal("expected bootstrap hook to be enabled")
	}
	if emitter != nil {
		t.Fatalf("emitter = %v, want nil", emitter)
	}
}

func TestRegistryRegister_RequiresVersion(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register(stubModule{id: "daggerheart", version: " "})
	if err != ErrSystemVersionRequired {
		t.Fatalf("expected ErrSystemVersionRequired, got %v", err)
	}
}

type errSentinel string

func (e errSentinel) Error() string { return string(e) }
