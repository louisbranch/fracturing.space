package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func adapterRegistryForStores(stores Stores) *systems.AdapterRegistry {
	registry := systems.NewAdapterRegistry()
	if stores.Daggerheart != nil {
		registry.Register(daggerheart.NewAdapter(stores.Daggerheart))
	}
	return registry
}
