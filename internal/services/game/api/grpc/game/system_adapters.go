package game

import (
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
)

// TryAdapterRegistryForSystemStores builds the adapter registry from the
// concrete system projection stores available to the core game transport.
func TryAdapterRegistryForSystemStores(stores SystemStores) (*bridge.AdapterRegistry, error) {
	return systemmanifest.AdapterRegistry(stores.Daggerheart)
}
