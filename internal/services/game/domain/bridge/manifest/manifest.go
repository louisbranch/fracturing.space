package manifest

import (
	"fmt"
	"strings"

	domainbridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	domainsystem "github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
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
	ID                  string
	Version             string
	BuildModule         func() domainsystem.Module
	BuildMetadataSystem func() domainbridge.GameSystem
	// BuildAdapter receives the full ProjectionStores so each system can extract
	// the store it needs. Return nil to skip adapter registration (e.g. when the
	// required store is absent).
	BuildAdapter func(ProjectionStores) domainbridge.Adapter
	// HasProfileSupport declares that this system participates in character
	// profile updates. When true, ValidateProfileAdapterCoverage asserts
	// the adapter implements bridge.ProfileAdapter at startup.
	HasProfileSupport bool
}

var builtInSystems = []SystemDescriptor{
	{
		ID:                  daggerheart.SystemID,
		Version:             strings.TrimSpace(daggerheart.SystemVersion),
		BuildModule:         func() domainsystem.Module { return daggerheart.NewModule() },
		BuildMetadataSystem: func() domainbridge.GameSystem { return daggerheart.NewRegistrySystem() },
		BuildAdapter: func(stores ProjectionStores) domainbridge.Adapter {
			if stores.Daggerheart == nil {
				return nil
			}
			return daggerheart.NewAdapter(stores.Daggerheart)
		},
		HasProfileSupport: true,
	},
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
func AdapterRegistry(stores ProjectionStores) (*domainbridge.AdapterRegistry, error) {
	return buildAdapterRegistry(stores)
}

// RebindAdapterRegistry creates a new adapter registry with swapped stores.
// It rebuilds adapters from the system descriptors using the provided stores,
// producing a fresh registry independent of the base. The base parameter is
// accepted (and validated as non-nil) to make call sites explicit about the
// intended pattern: build a base registry once at startup, then rebind
// per-transaction with transaction-scoped stores.
func RebindAdapterRegistry(base *domainbridge.AdapterRegistry, stores ProjectionStores) (*domainbridge.AdapterRegistry, error) {
	if base == nil {
		return nil, fmt.Errorf("base adapter registry is required for rebinding")
	}
	return buildAdapterRegistry(stores)
}

// ValidateProfileAdapterCoverage checks that every system descriptor declaring
// HasProfileSupport has an adapter that implements bridge.ProfileAdapter. This
// catches wiring bugs at startup instead of at the first profile update, where
// the missing interface would silently skip the system's profile projection.
func ValidateProfileAdapterCoverage(registry *domainbridge.AdapterRegistry) error {
	for _, d := range builtInSystems {
		if !d.HasProfileSupport {
			continue
		}
		adapter, ok := registry.GetOptional(d.ID, d.Version)
		if !ok {
			// Adapter not registered (e.g. store missing) — skip.
			continue
		}
		if _, ok := adapter.(domainbridge.ProfileAdapter); !ok {
			return fmt.Errorf("system %s@%s declares profile support but its adapter does not implement ProfileAdapter", d.ID, d.Version)
		}
	}
	return nil
}

// buildAdapterRegistry constructs an adapter registry from system descriptors.
func buildAdapterRegistry(stores ProjectionStores) (*domainbridge.AdapterRegistry, error) {
	registry := domainbridge.NewAdapterRegistry()
	for _, descriptor := range SystemDescriptors() {
		if descriptor.BuildAdapter == nil {
			continue
		}
		adapter := descriptor.BuildAdapter(stores)
		if adapter == nil {
			continue
		}
		if err := registry.Register(adapter); err != nil {
			return nil, fmt.Errorf("register adapter %s@%s: %w", descriptor.ID, descriptor.Version, err)
		}
	}
	return registry, nil
}
