package manifest

import (
	"fmt"
	"strings"

	domainsystem "github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
	domainsystems "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// ProjectionStores lists projection dependencies used by built-in system adapters.
type ProjectionStores struct {
	Daggerheart storage.DaggerheartStore
}

// SystemDescriptor declares one built-in system and how each startup surface should
// wire it. Keeping this list explicit makes add/remove operations discoverable
// for newcomers and reviewable in one file.
type SystemDescriptor struct {
	ID                   string
	Version              string
	BuildModule          func() domainsystem.Module
	BuildMetadataSystem  func() domainsystems.GameSystem
	BuildAdapter         func(storage.DaggerheartStore) domainsystems.Adapter
	RequiresAdapterStore bool
}

var builtInSystems = []SystemDescriptor{
	{
		ID:                   daggerheart.SystemID,
		Version:              strings.TrimSpace(daggerheart.SystemVersion),
		BuildModule:          func() domainsystem.Module { return daggerheart.NewModule() },
		BuildMetadataSystem:  func() domainsystems.GameSystem { return daggerheart.NewRegistrySystem() },
		BuildAdapter:         func(store storage.DaggerheartStore) domainsystems.Adapter { return daggerheart.NewAdapter(store) },
		RequiresAdapterStore: true,
	},
}

// SystemDescriptors returns the list of built-in system descriptors.
func SystemDescriptors() []SystemDescriptor {
	out := make([]SystemDescriptor, len(builtInSystems))
	copy(out, builtInSystems)
	return out
}

// Modules returns all built-in system modules.
func Modules() []domainsystem.Module {
	descriptors := SystemDescriptors()
	modules := make([]domainsystem.Module, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.BuildModule == nil {
			continue
		}
		module := descriptor.BuildModule()
		if module == nil {
			continue
		}
		modules = append(modules, module)
	}
	return modules
}

// MetadataSystems returns all built-in system metadata entries.
func MetadataSystems() []domainsystems.GameSystem {
	descriptors := SystemDescriptors()
	systemsList := make([]domainsystems.GameSystem, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.BuildMetadataSystem == nil {
			continue
		}
		if system := descriptor.BuildMetadataSystem(); system != nil {
			systemsList = append(systemsList, system)
		}
	}
	return systemsList
}

// AdapterRegistry returns a registry populated with built-in system adapters.
func AdapterRegistry(stores ProjectionStores) (*domainsystems.AdapterRegistry, error) {
	registry := domainsystems.NewAdapterRegistry()
	for _, descriptor := range SystemDescriptors() {
		if descriptor.BuildAdapter == nil {
			continue
		}
		if descriptor.RequiresAdapterStore && stores.Daggerheart == nil {
			continue
		}
		adapter := descriptor.BuildAdapter(stores.Daggerheart)
		if adapter == nil {
			continue
		}
		if err := registry.Register(adapter); err != nil {
			return nil, fmt.Errorf("register adapter %s@%s: %w", descriptor.ID, descriptor.Version, err)
		}
	}
	return registry, nil
}
