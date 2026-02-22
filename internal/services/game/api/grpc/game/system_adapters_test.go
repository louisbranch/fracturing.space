package game

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func TestAdapterRegistryForStoresEmpty(t *testing.T) {
	registry, err := TryAdapterRegistryForStores(Stores{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter := registry.Get(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, daggerheart.SystemVersion); adapter != nil {
		t.Fatal("expected no adapter when daggerheart store is nil")
	}
}

func TestAdapterRegistryForStoresRegistersDaggerheart(t *testing.T) {
	registry, err := TryAdapterRegistryForStores(Stores{Daggerheart: newFakeDaggerheartStore()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	adapter := registry.Get(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, daggerheart.SystemVersion)
	if adapter == nil {
		t.Fatal("expected daggerheart adapter to be registered")
	}
}
