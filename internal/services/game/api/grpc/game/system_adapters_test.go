package game

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
)

func TestAdapterRegistryForStoresEmpty(t *testing.T) {
	registry, err := TryAdapterRegistryForStores(Stores{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter := registry.Get(daggerheart.SystemID, daggerheart.SystemVersion); adapter != nil {
		t.Fatal("expected no adapter when daggerheart store is nil")
	}
}

func TestAdapterRegistryForStoresRegistersDaggerheart(t *testing.T) {
	registry, err := TryAdapterRegistryForStores(Stores{Daggerheart: newFakeDaggerheartStore()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	adapter := registry.Get(daggerheart.SystemID, daggerheart.SystemVersion)
	if adapter == nil {
		t.Fatal("expected daggerheart adapter to be registered")
	}
}
