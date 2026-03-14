package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
)

// TryAdapterRegistryForProjectionStores builds the adapter registry without
// panicking from the system projection-store bundle.
func TryAdapterRegistryForProjectionStores(stores systemmanifest.ProjectionStores) (*bridge.AdapterRegistry, error) {
	return systemmanifest.AdapterRegistry(stores)
}
