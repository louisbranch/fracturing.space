package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
)

func adapterRegistryForStores(stores Stores) *systems.AdapterRegistry {
	return systemmanifest.AdapterRegistry(systemmanifest.ProjectionStores{
		Daggerheart: stores.Daggerheart,
	})
}
