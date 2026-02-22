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
	registry := AdapterRegistry(ProjectionStores{
		Daggerheart: fakeDaggerheartStore{},
	})

	adapter := registry.Get(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, daggerheart.SystemVersion)
	if adapter == nil {
		t.Fatal("expected daggerheart adapter to be registered")
	}
}

func TestModulesHaveCorrespondingAdapters(t *testing.T) {
	modules := Modules()
	if len(modules) == 0 {
		t.Fatal("expected at least one registered module")
	}

	// Build adapter registry with all stores populated so adapters register.
	registry := AdapterRegistry(ProjectionStores{
		Daggerheart: fakeDaggerheartStore{},
	})

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
