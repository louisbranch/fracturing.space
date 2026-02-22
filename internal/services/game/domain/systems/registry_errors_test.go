package systems

import (
	"context"
	"errors"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type testAdapter struct {
	id      string
	version string
}

func (t *testAdapter) ID() string {
	return t.id
}

func (t *testAdapter) Version() string {
	return t.version
}

func (t *testAdapter) Apply(context.Context, event.Event) error {
	return nil
}

func (t *testAdapter) Snapshot(context.Context, string) (any, error) {
	return nil, nil
}

func (t *testAdapter) HandledTypes() []event.Type {
	return nil
}

type testGameSystem struct {
	id      commonv1.GameSystem
	version string
}

func (t *testGameSystem) ID() commonv1.GameSystem {
	return t.id
}

func (t *testGameSystem) Version() string {
	return t.version
}

func (t *testGameSystem) Name() string {
	return "test-system"
}

func (t *testGameSystem) RegistryMetadata() RegistryMetadata {
	return RegistryMetadata{}
}

func (t *testGameSystem) StateFactory() StateFactory {
	return nil
}

func (t *testGameSystem) OutcomeApplier() OutcomeApplier {
	return nil
}

func TestAdapterRegistryRegister_RequiresVersion(t *testing.T) {
	registry := NewAdapterRegistry()
	err := registry.Register(&testAdapter{
		id:      "daggerheart",
		version: "   ",
	})
	if !errors.Is(err, ErrAdapterVersionRequired) {
		t.Fatalf("expected ErrAdapterVersionRequired, got %v", err)
	}
}

func TestAdapterRegistryRegister_Duplicate(t *testing.T) {
	registry := NewAdapterRegistry()
	adapter := &testAdapter{
		id:      "daggerheart",
		version: "1.0.0",
	}
	if err := registry.Register(adapter); err != nil {
		t.Fatalf("first register: %v", err)
	}
	err := registry.Register(adapter)
	if !errors.Is(err, ErrAdapterAlreadyRegistered) {
		t.Fatalf("expected ErrAdapterAlreadyRegistered, got %v", err)
	}
}

func TestAdapterRegistryRegister_NilRegistry(t *testing.T) {
	var registry *AdapterRegistry
	err := registry.Register(&testAdapter{
		id:      "daggerheart",
		version: "1.0.0",
	})
	if !errors.Is(err, ErrAdapterRegistryNil) {
		t.Fatalf("expected ErrAdapterRegistryNil, got %v", err)
	}
}

func TestAdapterRegistryRegister_NilAdapter(t *testing.T) {
	registry := NewAdapterRegistry()
	err := registry.Register(nil)
	if !errors.Is(err, ErrAdapterRequired) {
		t.Fatalf("expected ErrAdapterRequired, got %v", err)
	}
}

func TestAdapterRegistryGetRequired_ReturnsErrorForUnknownSystem(t *testing.T) {
	registry := NewAdapterRegistry()
	_, err := registry.GetRequired("daggerheart", "1.0.0")
	if err == nil {
		t.Fatal("expected error for unknown system")
	}
	if !errors.Is(err, ErrAdapterNotFound) {
		t.Fatalf("expected ErrAdapterNotFound, got %v", err)
	}
}

func TestAdapterRegistryGetRequired_ReturnsAdapterWhenRegistered(t *testing.T) {
	registry := NewAdapterRegistry()
	adapter := &testAdapter{
		id:      "daggerheart",
		version: "1.0.0",
	}
	if err := registry.Register(adapter); err != nil {
		t.Fatalf("register: %v", err)
	}
	got, err := registry.GetRequired("daggerheart", "1.0.0")
	if err != nil {
		t.Fatalf("GetRequired: %v", err)
	}
	if got != adapter {
		t.Fatalf("GetRequired = %v, want %v", got, adapter)
	}
}

func TestAdapterRegistryGetRequired_NilRegistryReturnsError(t *testing.T) {
	var registry *AdapterRegistry
	_, err := registry.GetRequired("daggerheart", "1.0.0")
	if err == nil {
		t.Fatal("expected error for nil registry")
	}
	if !errors.Is(err, ErrAdapterRegistryNil) {
		t.Fatalf("expected ErrAdapterRegistryNil, got %v", err)
	}
}

func TestRegistryRegister_RequiresVersion(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register(&testGameSystem{
		id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		version: "   ",
	})
	if !errors.Is(err, ErrSystemVersionRequired) {
		t.Fatalf("expected ErrSystemVersionRequired, got %v", err)
	}
}

func TestRegistryRegister_Duplicate(t *testing.T) {
	registry := NewRegistry()
	system := &testGameSystem{
		id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		version: "1.0.0",
	}
	if err := registry.Register(system); err != nil {
		t.Fatalf("first register: %v", err)
	}
	err := registry.Register(system)
	if !errors.Is(err, ErrSystemAlreadyRegistered) {
		t.Fatalf("expected ErrSystemAlreadyRegistered, got %v", err)
	}
}

func TestRegistryRegister_NilRegistry(t *testing.T) {
	var registry *Registry
	err := registry.Register(&testGameSystem{
		id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		version: "1.0.0",
	})
	if !errors.Is(err, ErrSystemRegistryNil) {
		t.Fatalf("expected ErrSystemRegistryNil, got %v", err)
	}
}

func TestRegistryRegister_NilSystem(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register(nil)
	if !errors.Is(err, ErrSystemRequired) {
		t.Fatalf("expected ErrSystemRequired, got %v", err)
	}
}

func TestRegistryMustGetPanicsWhenMissing(t *testing.T) {
	registry := NewRegistry()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustGet() did not panic")
		}
		if err, ok := r.(error); !ok || !errors.Is(err, ErrSystemNotRegistered) {
			t.Fatalf("MustGet() panic = %v, want ErrSystemNotRegistered", r)
		}
	}()
	registry.MustGet(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
}

func TestRegistryGetOrError(t *testing.T) {
	registry := NewRegistry()
	_, err := registry.GetOrError(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	if !errors.Is(err, ErrSystemNotRegistered) {
		t.Fatalf("expected ErrSystemNotRegistered, got %v", err)
	}

	system := &testGameSystem{
		id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		version: "1.0",
	}
	if err := registry.Register(system); err != nil {
		t.Fatalf("register: %v", err)
	}
	got, err := registry.GetOrError(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	if err != nil {
		t.Fatalf("GetOrError() unexpected error: %v", err)
	}
	if got != system {
		t.Fatalf("GetOrError() = %v, want %v", got, system)
	}
}
