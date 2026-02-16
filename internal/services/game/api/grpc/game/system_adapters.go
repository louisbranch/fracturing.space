package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func adapterRegistryForStores(stores Stores) *systems.AdapterRegistry {
	registry := systems.NewAdapterRegistry()
	if stores.Daggerheart != nil {
		// Built-in adapters are expected to register cleanly; on error we return
		// an empty/partial registry so callers can still surface missing-adapter
		// failures through normal apply paths.
		if err := registry.Register(daggerheart.NewAdapter(stores.Daggerheart)); err != nil {
			return registry
		}
	}
	return registry
}
