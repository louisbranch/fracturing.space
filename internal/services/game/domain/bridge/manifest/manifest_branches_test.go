package manifest

import (
	"testing"

	domainbridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	domainsystem "github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func TestRebindAdapterRegistry_RequiresBaseRegistry(t *testing.T) {
	_, err := RebindAdapterRegistry(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil base registry")
	}
	if got := err.Error(); got != "base adapter registry is required for rebinding" {
		t.Fatalf("error = %q, want %q", got, "base adapter registry is required for rebinding")
	}
}

func TestModules_SkipsNilBuildersAndNilModules(t *testing.T) {
	orig := builtInSystems
	builtInSystems = []SystemDescriptor{
		{
			ID:                  "skip-nil-builder",
			Version:             "v1",
			BuildModule:         nil,
			BuildMetadataSystem: func() domainbridge.GameSystem { return nil },
			BuildAdapter:        func(any) domainbridge.Adapter { return nil },
		},
		{
			ID:                  "skip-nil-module",
			Version:             "v1",
			BuildModule:         func() domainsystem.Module { return nil },
			BuildMetadataSystem: func() domainbridge.GameSystem { return nil },
			BuildAdapter:        func(any) domainbridge.Adapter { return nil },
		},
		{
			ID:                  "keep-module",
			Version:             "v1",
			BuildModule:         func() domainsystem.Module { return moduleStub{id: "keep-module", version: "v1"} },
			BuildMetadataSystem: func() domainbridge.GameSystem { return nil },
			BuildAdapter:        func(any) domainbridge.Adapter { return nil },
		},
	}
	defer func() { builtInSystems = orig }()

	modules := Modules()
	if len(modules) != 1 {
		t.Fatalf("Modules() len = %d, want 1", len(modules))
	}
	if modules[0].ID() != "keep-module" {
		t.Fatalf("module id = %q, want %q", modules[0].ID(), "keep-module")
	}
}

func TestMetadataSystems_SkipsNilBuildersAndNilSystems(t *testing.T) {
	orig := builtInSystems
	builtInSystems = []SystemDescriptor{
		{
			ID:                  "skip-nil-builder",
			Version:             "v1",
			BuildModule:         func() domainsystem.Module { return nil },
			BuildMetadataSystem: nil,
			BuildAdapter:        func(any) domainbridge.Adapter { return nil },
		},
		{
			ID:                  "skip-nil-system",
			Version:             "v1",
			BuildModule:         func() domainsystem.Module { return nil },
			BuildMetadataSystem: func() domainbridge.GameSystem { return nil },
			BuildAdapter:        func(any) domainbridge.Adapter { return nil },
		},
		{
			ID:      "keep-system",
			Version: "v1",
			BuildModule: func() domainsystem.Module {
				return nil
			},
			BuildMetadataSystem: func() domainbridge.GameSystem {
				return metadataSystemStub{id: domainbridge.SystemIDDaggerheart, version: "v1"}
			},
			BuildAdapter: func(any) domainbridge.Adapter { return nil },
		},
	}
	defer func() { builtInSystems = orig }()

	systems := MetadataSystems()
	if len(systems) != 1 {
		t.Fatalf("MetadataSystems() len = %d, want 1", len(systems))
	}
	if systems[0].Version() != "v1" {
		t.Fatalf("system version = %q, want %q", systems[0].Version(), "v1")
	}
}

type moduleStub struct {
	id      string
	version string
}

func (m moduleStub) ID() string                               { return m.id }
func (m moduleStub) Version() string                          { return m.version }
func (m moduleStub) RegisterCommands(*command.Registry) error { return nil }
func (m moduleStub) RegisterEvents(*event.Registry) error     { return nil }
func (m moduleStub) EmittableEventTypes() []event.Type        { return nil }
func (m moduleStub) Decider() domainsystem.Decider            { return nil }
func (m moduleStub) Folder() domainsystem.Folder              { return nil }
func (m moduleStub) StateFactory() domainsystem.StateFactory  { return nil }

type metadataSystemStub struct {
	id      domainbridge.SystemID
	version string
}

func (m metadataSystemStub) ID() domainbridge.SystemID { return m.id }
func (m metadataSystemStub) Version() string           { return m.version }
func (m metadataSystemStub) Name() string              { return "stub" }
func (m metadataSystemStub) RegistryMetadata() domainbridge.RegistryMetadata {
	return domainbridge.RegistryMetadata{}
}
func (m metadataSystemStub) StateHandlerFactory() domainbridge.StateHandlerFactory { return nil }
func (m metadataSystemStub) OutcomeApplier() domainbridge.OutcomeApplier           { return nil }
