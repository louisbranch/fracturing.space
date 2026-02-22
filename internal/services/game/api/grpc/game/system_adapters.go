package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
)

func adapterRegistryForStores(stores Stores) *systems.AdapterRegistry {
	registry, err := TryAdapterRegistryForStores(stores)
	if err != nil {
		panic(err)
	}
	return registry
}

// TryAdapterRegistryForStores builds the adapter registry without panicking.
// Use this at startup or tests that validate registration health.
func TryAdapterRegistryForStores(stores Stores) (*systems.AdapterRegistry, error) {
	return systemmanifest.AdapterRegistry(systemmanifest.ProjectionStores{
		Daggerheart: stores.Daggerheart,
	})
}
