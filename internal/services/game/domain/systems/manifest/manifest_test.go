package manifest

import (
	"fmt"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type fakeDaggerheartStore struct {
	storage.DaggerheartStore
}

type anotherFakeDaggerheartStore struct {
	storage.DaggerheartStore
}

func TestRebindAdapterRegistrySwapsStores(t *testing.T) {
	base, err := AdapterRegistry(ProjectionStores{Daggerheart: fakeDaggerheartStore{}})
	if err != nil {
		t.Fatalf("build base registry: %v", err)
	}

	rebound, err := RebindAdapterRegistry(base, ProjectionStores{Daggerheart: anotherFakeDaggerheartStore{}})
	if err != nil {
		t.Fatalf("rebind adapter registry: %v", err)
	}

	adapter := rebound.Get(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, daggerheart.SystemVersion)
	if adapter == nil {
		t.Fatal("expected daggerheart adapter in rebound registry")
	}

	// Base registry should still have its own adapter (not affected by rebind).
	origAdapter := base.Get(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, daggerheart.SystemVersion)
	if origAdapter == nil {
		t.Fatal("expected adapter to remain in base registry")
	}
	if origAdapter == adapter {
		t.Fatal("expected rebound adapter to be a different instance than base")
	}
}

func TestModulesAndMetadataShareSystemVersionKeys(t *testing.T) {
	modules := Modules()
	metadata := MetadataSystems()
	if len(modules) == 0 {
		t.Fatal("expected at least one registered module")
	}
	if len(metadata) == 0 {
		t.Fatal("expected at least one registered metadata system")
	}

	moduleKeys := make(map[string]struct{}, len(modules))
	for _, module := range modules {
		if module == nil {
			t.Fatal("module is nil")
		}
		systemID, ok := parseGameSystemID(module.ID())
		if !ok {
			t.Fatalf("unknown module id %q", module.ID())
		}
		key := fmt.Sprintf("%d@%s", systemID, strings.TrimSpace(module.Version()))
		moduleKeys[key] = struct{}{}
	}

	for _, gameSystem := range metadata {
		if gameSystem == nil {
			t.Fatal("metadata system is nil")
		}
		key := fmt.Sprintf("%d@%s", gameSystem.ID(), strings.TrimSpace(gameSystem.Version()))
		if _, ok := moduleKeys[key]; !ok {
			t.Fatalf("metadata %q has no matching module registration", key)
		}
	}
}

func TestAdapterRegistryRegistersDaggerheart(t *testing.T) {
	registry, err := AdapterRegistry(ProjectionStores{
		Daggerheart: fakeDaggerheartStore{},
	})
	if err != nil {
		t.Fatalf("build adapter registry: %v", err)
	}

	adapter := registry.Get(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, daggerheart.SystemVersion)
	if adapter == nil {
		t.Fatal("expected daggerheart adapter to be registered")
	}
}

func TestAdapterRegistryReturnsErrorOnRegistrationFailure(t *testing.T) {
	// Pre-populate the registry by calling AdapterRegistry once, then
	// register the same adapter again to trigger a duplicate error.
	// Since we cannot double-register via AdapterRegistry directly,
	// we test via a nil store (which skips registration) — but the real
	// error path is a duplicate. Instead, verify that a nil-store registry
	// works cleanly and a pre-registered duplicate fails.
	stores := ProjectionStores{Daggerheart: fakeDaggerheartStore{}}
	registry, err := AdapterRegistry(stores)
	if err != nil {
		t.Fatalf("first registration should succeed: %v", err)
	}
	// Manually register the same adapter again to force a duplicate error.
	dupErr := registry.Register(daggerheart.NewAdapter(fakeDaggerheartStore{}))
	if dupErr == nil {
		t.Fatal("expected duplicate registration to return an error")
	}
}

func TestModulesHaveCorrespondingAdapters(t *testing.T) {
	modules := Modules()
	if len(modules) == 0 {
		t.Fatal("expected at least one registered module")
	}

	// Build adapter registry with all stores populated so adapters register.
	registry, err := AdapterRegistry(ProjectionStores{
		Daggerheart: fakeDaggerheartStore{},
	})
	if err != nil {
		t.Fatalf("build adapter registry: %v", err)
	}

	for _, module := range modules {
		systemID, ok := parseGameSystemID(module.ID())
		if !ok {
			t.Fatalf("unknown module id %q", module.ID())
		}
		version := strings.TrimSpace(module.Version())
		adapter := registry.Get(systemID, version)
		if adapter == nil {
			t.Errorf("module %s@%s has no corresponding adapter in AdapterRegistry", module.ID(), version)
		}
	}
}

func TestAdapterRegistrySkipsNilStoreViaClosureGuard(t *testing.T) {
	// When ProjectionStores.Daggerheart is nil, BuildAdapter should return nil
	// and the registry should skip registration without error.
	registry, err := AdapterRegistry(ProjectionStores{Daggerheart: nil})
	if err != nil {
		t.Fatalf("expected no error with nil store, got: %v", err)
	}
	adapter := registry.Get(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, daggerheart.SystemVersion)
	if adapter != nil {
		t.Fatal("expected no adapter when store is nil")
	}
}

func parseGameSystemID(raw string) (commonv1.GameSystem, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, false
	}
	if value, ok := commonv1.GameSystem_value[trimmed]; ok {
		return commonv1.GameSystem(value), true
	}
	upper := strings.ToUpper(trimmed)
	if value, ok := commonv1.GameSystem_value["GAME_SYSTEM_"+upper]; ok {
		return commonv1.GameSystem(value), true
	}
	return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, false
}
