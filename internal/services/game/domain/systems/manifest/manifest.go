package manifest

import (
	domainsystem "github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
	domainsystems "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// ProjectionStores lists projection dependencies used by built-in system adapters.
type ProjectionStores struct {
	Daggerheart storage.DaggerheartStore
}

// Modules returns all built-in system modules.
func Modules() []domainsystem.Module {
	return []domainsystem.Module{
		daggerheart.NewModule(),
	}
}

// MetadataSystems returns all built-in system metadata entries.
func MetadataSystems() []domainsystems.GameSystem {
	return []domainsystems.GameSystem{
		daggerheart.NewRegistrySystem(),
	}
}

// AdapterRegistry returns a registry populated with built-in system adapters.
func AdapterRegistry(stores ProjectionStores) *domainsystems.AdapterRegistry {
	registry := domainsystems.NewAdapterRegistry()
	if stores.Daggerheart != nil {
		if err := registry.Register(daggerheart.NewAdapter(stores.Daggerheart)); err != nil {
			return registry
		}
	}
	return registry
}
