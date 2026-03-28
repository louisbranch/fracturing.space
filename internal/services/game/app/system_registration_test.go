package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
)

func TestLoadSystemRegistrationSnapshotStaysManifestDerived(t *testing.T) {
	snapshot := loadSystemRegistrationSnapshot()

	manifestModules := systemmanifest.Modules()
	registeredModules := snapshot.modulesCopy()
	if len(registeredModules) != len(manifestModules) {
		t.Fatalf("registered module count = %d, want %d", len(registeredModules), len(manifestModules))
	}
	for i := range manifestModules {
		if registeredModules[i].ID() != manifestModules[i].ID() || registeredModules[i].Version() != manifestModules[i].Version() {
			t.Fatalf(
				"registered module[%d] = %s@%s, want %s@%s",
				i,
				registeredModules[i].ID(),
				registeredModules[i].Version(),
				manifestModules[i].ID(),
				manifestModules[i].Version(),
			)
		}
	}

	manifestMetadata := systemmanifest.MetadataSystems()
	registeredMetadata := snapshot.metadataSystemsCopy()
	if len(registeredMetadata) != len(manifestMetadata) {
		t.Fatalf("registered metadata count = %d, want %d", len(registeredMetadata), len(manifestMetadata))
	}
	for i := range manifestMetadata {
		if registeredMetadata[i].ID() != manifestMetadata[i].ID() || registeredMetadata[i].Version() != manifestMetadata[i].Version() {
			t.Fatalf(
				"registered metadata[%d] = %s@%s, want %s@%s",
				i,
				registeredMetadata[i].ID(),
				registeredMetadata[i].Version(),
				manifestMetadata[i].ID(),
				manifestMetadata[i].Version(),
			)
		}
	}
}

func TestSystemRegistrationSnapshotBuildsMetadataRegistry(t *testing.T) {
	snapshot := loadSystemRegistrationSnapshot()

	registry, err := snapshot.buildMetadataRegistry()
	if err != nil {
		t.Fatalf("build metadata registry: %v", err)
	}

	metadataSystems := snapshot.metadataSystemsCopy()
	if got, want := len(registry.List()), len(metadataSystems); got != want {
		t.Fatalf("registry metadata count = %d, want %d", got, want)
	}
}

func TestValidateSystemRegistrationParity(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		modules := []fakeSystemModule{
			{id: "DAGGERHEART", version: "v1"},
		}
		registry := bridge.NewMetadataRegistry()
		if err := registry.Register(fakeGameSystem{
			id:      bridge.SystemIDDaggerheart,
			version: "v1",
		}); err != nil {
			t.Fatalf("register metadata system: %v", err)
		}
		adapters := bridge.NewAdapterRegistry()
		if err := adapters.Register(fakeSystemAdapter{
			id:      "DAGGERHEART",
			version: "v1",
		}); err != nil {
			t.Fatalf("register adapter: %v", err)
		}
		if err := validateSystemRegistrationParity(asModules(modules), registry, adapters); err != nil {
			t.Fatalf("validate parity: %v", err)
		}
	})

	t.Run("missing adapter", func(t *testing.T) {
		modules := []fakeSystemModule{
			{id: "DAGGERHEART", version: "v1"},
		}
		registry := bridge.NewMetadataRegistry()
		if err := registry.Register(fakeGameSystem{
			id:      bridge.SystemIDDaggerheart,
			version: "v1",
		}); err != nil {
			t.Fatalf("register metadata system: %v", err)
		}
		adapters := bridge.NewAdapterRegistry()

		err := validateSystemRegistrationParity(asModules(modules), registry, adapters)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "adapter") {
			t.Fatalf("error = %q, want adapter detail", err.Error())
		}
	})

	t.Run("metadata without module", func(t *testing.T) {
		registry := bridge.NewMetadataRegistry()
		if err := registry.Register(fakeGameSystem{
			id:      bridge.SystemIDDaggerheart,
			version: "v1",
		}); err != nil {
			t.Fatalf("register metadata system: %v", err)
		}
		adapters := bridge.NewAdapterRegistry()
		if err := adapters.Register(fakeSystemAdapter{
			id:      "DAGGERHEART",
			version: "v1",
		}); err != nil {
			t.Fatalf("register adapter: %v", err)
		}

		err := validateSystemRegistrationParity(nil, registry, adapters)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, errSystemModuleRegistryMismatch) {
			t.Fatalf("error = %v, want %v", err, errSystemModuleRegistryMismatch)
		}
	})
}

func asModules(modules []fakeSystemModule) []module.Module {
	out := make([]module.Module, 0, len(modules))
	for _, module := range modules {
		out = append(out, module)
	}
	return out
}

func manifestSystemModulesForTest() []module.Module {
	return loadSystemRegistrationSnapshot().modulesCopy()
}
