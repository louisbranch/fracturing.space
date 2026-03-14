package game

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
)

func TestAdapterRegistryForProjectionStoresEmpty(t *testing.T) {
	registry, err := TryAdapterRegistryForProjectionStores(systemmanifest.ProjectionStores{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Has(daggerheart.SystemID, daggerheart.SystemVersion) {
		t.Fatal("expected no adapter when daggerheart store is nil")
	}
}

func TestAdapterRegistryForProjectionStoresRegistersDaggerheart(t *testing.T) {
	registry, err := TryAdapterRegistryForProjectionStores(systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !registry.Has(daggerheart.SystemID, daggerheart.SystemVersion) {
		t.Fatal("expected daggerheart adapter to be registered")
	}
}
