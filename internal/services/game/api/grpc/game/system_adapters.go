package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
)

func adapterRegistryForStores(stores Stores) *bridge.AdapterRegistry {
	registry, err := TryAdapterRegistryForStores(stores)
	if err != nil {
		panic(err)
	}
	return registry
}

// TryAdapterRegistryForStores builds the adapter registry without panicking.
// Use this at startup or tests that validate registration health.
func TryAdapterRegistryForStores(stores Stores) (*bridge.AdapterRegistry, error) {
	return systemmanifest.AdapterRegistry(stores.SystemStores)
}
