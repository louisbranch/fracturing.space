package game

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
)

func TestAdapterRegistryForSystemStoresEmpty(t *testing.T) {
	registry, err := TryAdapterRegistryForSystemStores(SystemStores{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Has(daggerheart.SystemID, daggerheart.SystemVersion) {
		t.Fatal("expected no adapter when daggerheart store is nil")
	}
}

func TestAdapterRegistryForSystemStoresRegistersDaggerheart(t *testing.T) {
	registry, err := TryAdapterRegistryForSystemStores(SystemStores{Daggerheart: gametest.NewFakeDaggerheartStore()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !registry.Has(daggerheart.SystemID, daggerheart.SystemVersion) {
		t.Fatal("expected daggerheart adapter to be registered")
	}
}
