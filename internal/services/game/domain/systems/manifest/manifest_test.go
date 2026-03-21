package manifest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	domainsystem "github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	domainbridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

type fakeDaggerheartStore struct {
	projectionstore.Store
}

func (fakeDaggerheartStore) ListDaggerheartCharacterProfiles(context.Context, string, int, string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	return projectionstore.DaggerheartCharacterProfilePage{}, nil
}

type anotherFakeDaggerheartStore struct {
	projectionstore.Store
}

func (anotherFakeDaggerheartStore) ListDaggerheartCharacterProfiles(context.Context, string, int, string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	return projectionstore.DaggerheartCharacterProfilePage{}, nil
}

type daggerheartProjectionStoreProviderStub struct {
	store projectionstore.Store
}

func (p daggerheartProjectionStoreProviderStub) DaggerheartProjectionStore() projectionstore.Store {
	return p.store
}

func TestRebindAdapterRegistrySwapsStores(t *testing.T) {
	base, err := AdapterRegistry(fakeDaggerheartStore{})
	if err != nil {
		t.Fatalf("build base registry: %v", err)
	}

	rebound, err := RebindAdapterRegistry(base, anotherFakeDaggerheartStore{})
	if err != nil {
		t.Fatalf("rebind adapter registry: %v", err)
	}

	adapter, ok := rebound.GetOptional(daggerheart.SystemID, daggerheart.SystemVersion)
	if !ok {
		t.Fatal("expected daggerheart adapter in rebound registry")
	}

	// Base registry should still have its own adapter (not affected by rebind).
	origAdapter, ok := base.GetOptional(daggerheart.SystemID, daggerheart.SystemVersion)
	if !ok {
		t.Fatal("expected adapter to remain in base registry")
	}
	if origAdapter == adapter {
		t.Fatal("expected rebound adapter to be a different instance than base")
	}
}

func TestModulesAndMetadataShareSystemVersionKeys(t *testing.T) {
	modules := Modules()
	metadata := MetadataSystems()
	if len(modules) == 0 {
		t.Fatal("expected at least one registered module")
	}
	if len(metadata) == 0 {
		t.Fatal("expected at least one registered metadata system")
	}

	moduleKeys := make(map[string]struct{}, len(modules))
	for _, module := range modules {
		if module == nil {
			t.Fatal("module is nil")
		}
		systemID, ok := parseGameSystemID(module.ID())
		if !ok {
			t.Fatalf("unknown module id %q", module.ID())
		}
		key := fmt.Sprintf("%s@%s", systemID, strings.TrimSpace(module.Version()))
		moduleKeys[key] = struct{}{}
	}

	for _, gameSystem := range metadata {
		if gameSystem == nil {
			t.Fatal("metadata system is nil")
		}
		key := fmt.Sprintf("%s@%s", gameSystem.ID(), strings.TrimSpace(gameSystem.Version()))
		if _, ok := moduleKeys[key]; !ok {
			t.Fatalf("metadata %q has no matching module registration", key)
		}
	}
}

func TestAdapterRegistryRegistersDaggerheart(t *testing.T) {
	registry, err := AdapterRegistry(fakeDaggerheartStore{})
	if err != nil {
		t.Fatalf("build adapter registry: %v", err)
	}

	if !registry.Has(daggerheart.SystemID, daggerheart.SystemVersion) {
		t.Fatal("expected daggerheart adapter to be registered")
	}
}

func TestAdapterRegistryReturnsErrorOnRegistrationFailure(t *testing.T) {
	// Pre-populate the registry by calling AdapterRegistry once, then
	// register the same adapter again to trigger a duplicate error.
	// Since we cannot double-register via AdapterRegistry directly,
	// we test via a nil store (which skips registration) — but the real
	// error path is a duplicate. Instead, verify that a nil-store registry
	// works cleanly and a pre-registered duplicate fails.
	registry, err := AdapterRegistry(fakeDaggerheartStore{})
	if err != nil {
		t.Fatalf("first registration should succeed: %v", err)
	}
	// Manually register the same adapter again to force a duplicate error.
	dupErr := registry.Register(daggerheart.NewAdapter(fakeDaggerheartStore{}))
	if dupErr == nil {
		t.Fatal("expected duplicate registration to return an error")
	}
}

func TestModulesHaveCorrespondingAdapters(t *testing.T) {
	modules := Modules()
	if len(modules) == 0 {
		t.Fatal("expected at least one registered module")
	}

	// Build adapter registry with all stores populated so adapters register.
	registry, err := AdapterRegistry(fakeDaggerheartStore{})
	if err != nil {
		t.Fatalf("build adapter registry: %v", err)
	}

	for _, module := range modules {
		moduleID := strings.TrimSpace(module.ID())
		version := strings.TrimSpace(module.Version())
		if !registry.Has(moduleID, version) {
			t.Errorf("module %s@%s has no corresponding adapter in AdapterRegistry", moduleID, version)
		}
	}
}

func TestAdapterRegistrySkipsNilStoreViaClosureGuard(t *testing.T) {
	// When the concrete store source does not expose a Daggerheart projection
	// store, BuildAdapter should return nil and the registry should skip
	// registration without error.
	registry, err := AdapterRegistry(nil)
	if err != nil {
		t.Fatalf("expected no error with nil store, got: %v", err)
	}
	if registry.Has(daggerheart.SystemID, daggerheart.SystemVersion) {
		t.Fatal("expected no adapter when store is nil")
	}
}

func TestValidateSystemDescriptors_PassesForBuiltIns(t *testing.T) {
	if err := ValidateSystemDescriptors(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateSystemDescriptors_RejectsNilBuildModule(t *testing.T) {
	orig := builtInSystems
	builtInSystems = []SystemDescriptor{{
		ID:                  "test",
		Version:             "v1",
		BuildModule:         nil,
		BuildMetadataSystem: func() domainbridge.GameSystem { return nil },
		BuildAdapter:        func(any) domainbridge.Adapter { return nil },
	}}
	defer func() { builtInSystems = orig }()

	err := ValidateSystemDescriptors()
	if err == nil {
		t.Fatal("expected error for nil BuildModule")
	}
	if !strings.Contains(err.Error(), "BuildModule") {
		t.Fatalf("expected error to mention BuildModule, got: %v", err)
	}
}

func TestValidateSystemDescriptors_RejectsNilBuildMetadataSystem(t *testing.T) {
	orig := builtInSystems
	builtInSystems = []SystemDescriptor{{
		ID:                  "test",
		Version:             "v1",
		BuildModule:         func() domainsystem.Module { return nil },
		BuildMetadataSystem: nil,
		BuildAdapter:        func(any) domainbridge.Adapter { return nil },
	}}
	defer func() { builtInSystems = orig }()

	err := ValidateSystemDescriptors()
	if err == nil {
		t.Fatal("expected error for nil BuildMetadataSystem")
	}
	if !strings.Contains(err.Error(), "BuildMetadataSystem") {
		t.Fatalf("expected error to mention BuildMetadataSystem, got: %v", err)
	}
}

func TestValidateSystemDescriptors_RejectsNilBuildAdapter(t *testing.T) {
	orig := builtInSystems
	builtInSystems = []SystemDescriptor{{
		ID:                  "test",
		Version:             "v1",
		BuildModule:         func() domainsystem.Module { return nil },
		BuildMetadataSystem: func() domainbridge.GameSystem { return nil },
		BuildAdapter:        nil,
	}}
	defer func() { builtInSystems = orig }()

	err := ValidateSystemDescriptors()
	if err == nil {
		t.Fatal("expected error for nil BuildAdapter")
	}
	if !strings.Contains(err.Error(), "BuildAdapter") {
		t.Fatalf("expected error to mention BuildAdapter, got: %v", err)
	}
}

// noProfileAdapter is a minimal adapter that does NOT implement ProfileAdapter.
type noProfileAdapter struct {
	id      string
	version string
}

func (a noProfileAdapter) ID() string                                        { return a.id }
func (a noProfileAdapter) Version() string                                   { return a.version }
func (a noProfileAdapter) Apply(_ context.Context, _ event.Event) error      { return nil }
func (a noProfileAdapter) Snapshot(_ context.Context, _ string) (any, error) { return nil, nil }
func (a noProfileAdapter) HandledTypes() []event.Type                        { return nil }

func TestDaggerheartProjectionStoreFromSource_PopulatesDaggerheart(t *testing.T) {
	store := fakeDaggerheartStore{}
	if got := daggerheartProjectionStoreFromSource(store); got == nil {
		t.Fatal("expected Daggerheart store to be populated")
	}
}

func TestDaggerheartProjectionStoreFromSource_PrefersProvider(t *testing.T) {
	provided := anotherFakeDaggerheartStore{}
	got := daggerheartProjectionStoreFromSource(daggerheartProjectionStoreProviderStub{store: provided})
	if got != provided {
		t.Fatal("expected explicit provider store to be used")
	}
}

func TestDaggerheartProjectionStoreFromSource_NilForNonImplementor(t *testing.T) {
	got := daggerheartProjectionStoreFromSource("not a store")
	if got != nil {
		t.Fatal("expected Daggerheart store to be nil for non-implementor")
	}
}

func TestDaggerheartProjectionStoreFromSource_NilInput(t *testing.T) {
	got := daggerheartProjectionStoreFromSource(nil)
	if got != nil {
		t.Fatal("expected Daggerheart store to be nil for nil input")
	}
}

func parseGameSystemID(raw string) (domainbridge.SystemID, bool) {
	return domainbridge.NormalizeSystemID(raw)
}
