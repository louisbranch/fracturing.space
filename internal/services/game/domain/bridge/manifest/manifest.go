package manifest

import (
	"fmt"
	"strings"

	domainbridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	domainsystem "github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

type daggerheartProjectionStoreProvider interface {
	DaggerheartProjectionStore() projectionstore.Store
}

// SystemDescriptor declares one built-in system and how each startup surface should
// wire it. Keeping this list explicit makes add/remove operations discoverable
// for newcomers and reviewable in one file.
type SystemDescriptor struct {
	ID                  string
	Version             string
	BuildModule         func() domainsystem.Module
	BuildMetadataSystem func() domainbridge.GameSystem
	// BuildAdapter receives the concrete store source available at startup or
	// replay time. Each system owns any extraction logic needed to obtain its
	// projection backend from that source. Return nil to skip adapter
	// registration when the required store is absent.
	BuildAdapter func(storeSource any) domainbridge.Adapter
}

var builtInSystems = []SystemDescriptor{
	{
		ID:                  daggerheart.SystemID,
		Version:             strings.TrimSpace(daggerheart.SystemVersion),
		BuildModule:         func() domainsystem.Module { return daggerheart.NewModule() },
		BuildMetadataSystem: func() domainbridge.GameSystem { return daggerheart.NewRegistrySystem() },
		BuildAdapter:        daggerheartAdapterFromStoreSource,
	},
}

func daggerheartAdapterFromStoreSource(storeSource any) domainbridge.Adapter {
	store := daggerheartProjectionStoreFromSource(storeSource)
	if store == nil {
		return nil
	}
	return daggerheart.NewAdapter(store)
}

func daggerheartProjectionStoreFromSource(storeSource any) projectionstore.Store {
	if provider, ok := storeSource.(daggerheartProjectionStoreProvider); ok {
		return provider.DaggerheartProjectionStore()
	}
	store, _ := storeSource.(projectionstore.Store)
	return store
}

// ValidateSystemDescriptors verifies that every built-in system descriptor
// has non-nil builders. A nil builder causes silent degradation at runtime
// instead of a clear startup failure.
func ValidateSystemDescriptors() error {
	for _, d := range builtInSystems {
		label := d.ID + "@" + d.Version
		if d.BuildModule == nil {
			return fmt.Errorf("system %s has nil BuildModule", label)
		}
		if d.BuildMetadataSystem == nil {
			return fmt.Errorf("system %s has nil BuildMetadataSystem", label)
		}
		if d.BuildAdapter == nil {
			return fmt.Errorf("system %s has nil BuildAdapter", label)
		}
	}
	return nil
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
func MetadataSystems() []domainbridge.GameSystem {
	descriptors := SystemDescriptors()
	systemsList := make([]domainbridge.GameSystem, 0, len(descriptors))
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
// It returns an error if any adapter registration fails, turning silent runtime
// failures into startup failures.
func AdapterRegistry(storeSource any) (*domainbridge.AdapterRegistry, error) {
	return buildAdapterRegistry(storeSource)
}

// RebindAdapterRegistry creates a new adapter registry with swapped stores.
// It rebuilds adapters from the system descriptors using the provided stores,
// producing a fresh registry independent of the base. The base parameter is
// accepted (and validated as non-nil) to make call sites explicit about the
// intended pattern: build a base registry once at startup, then rebind
// per-transaction with transaction-scoped stores.
func RebindAdapterRegistry(base *domainbridge.AdapterRegistry, storeSource any) (*domainbridge.AdapterRegistry, error) {
	if base == nil {
		return nil, fmt.Errorf("base adapter registry is required for rebinding")
	}
	return buildAdapterRegistry(storeSource)
}

// buildAdapterRegistry constructs an adapter registry from system descriptors.
func buildAdapterRegistry(storeSource any) (*domainbridge.AdapterRegistry, error) {
	registry := domainbridge.NewAdapterRegistry()
	for _, descriptor := range SystemDescriptors() {
		if descriptor.BuildAdapter == nil {
			continue
		}
		adapter := descriptor.BuildAdapter(storeSource)
		if adapter == nil {
			continue
		}
		if err := registry.Register(adapter); err != nil {
			return nil, fmt.Errorf("register adapter %s@%s: %w", descriptor.ID, descriptor.Version, err)
		}
	}
	return registry, nil
}
