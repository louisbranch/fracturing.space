package game

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
)

func TestAdapterRegistryForStoresEmpty(t *testing.T) {
	registry, err := TryAdapterRegistryForStores(Stores{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Has(daggerheart.SystemID, daggerheart.SystemVersion) {
		t.Fatal("expected no adapter when daggerheart store is nil")
	}
}

func TestAdapterRegistryForStoresRegistersDaggerheart(t *testing.T) {
	registry, err := TryAdapterRegistryForStores(Stores{SystemStores: systemmanifest.ProjectionStores{Daggerheart: newFakeDaggerheartStore()}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !registry.Has(daggerheart.SystemID, daggerheart.SystemVersion) {
		t.Fatal("expected daggerheart adapter to be registered")
	}
}
