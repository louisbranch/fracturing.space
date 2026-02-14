package systems

import (
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

type testSystem struct {
	id      commonv1.GameSystem
	version string
}

func (t *testSystem) ID() commonv1.GameSystem {
	return t.id
}

func (t *testSystem) Version() string {
	return t.version
}

func (t *testSystem) Name() string {
	return "Test"
}

func (t *testSystem) RegistryMetadata() RegistryMetadata {
	return RegistryMetadata{}
}

func (t *testSystem) StateFactory() StateFactory {
	return nil
}

func (t *testSystem) OutcomeApplier() OutcomeApplier {
	return nil
}

func TestRegistryDefaultsToFirstVersion(t *testing.T) {
	registry := NewRegistry()
	primary := &testSystem{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: "1.0.0"}
	secondary := &testSystem{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: "1.1.0"}

	registry.Register(primary)
	registry.Register(secondary)

	if got := registry.Get(primary.ID()); got != primary {
		t.Fatalf("Get default = %v, want primary", got)
	}
	if got := registry.DefaultVersion(primary.ID()); got != "1.0.0" {
		t.Fatalf("DefaultVersion = %q, want %q", got, "1.0.0")
	}
	if got := registry.GetVersion(primary.ID(), "1.1.0"); got != secondary {
		t.Fatalf("GetVersion = %v, want secondary", got)
	}
}

func TestRegistryDefaultVersionUnknownSystem(t *testing.T) {
	registry := NewRegistry()
	if got := registry.DefaultVersion(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART); got != "" {
		t.Fatalf("DefaultVersion = %q, want empty", got)
	}
}

func TestRegistryRejectsEmptyVersion(t *testing.T) {
	registry := NewRegistry()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for empty version")
		}
	}()

	registry.Register(&testSystem{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: ""})
}

func TestRegistryRejectsDuplicateVersion(t *testing.T) {
	registry := NewRegistry()
	primary := &testSystem{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: "1.0.0"}
	registry.Register(primary)

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for duplicate version")
		}
	}()

	registry.Register(primary)
}

func TestRegistryGetVersionWhitespace(t *testing.T) {
	registry := NewRegistry()
	primary := &testSystem{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: "1.0.0"}
	registry.Register(primary)

	// Whitespace version should fallback to default
	if got := registry.GetVersion(primary.ID(), "  "); got != primary {
		t.Fatalf("GetVersion with whitespace = %v, want primary", got)
	}
}

func TestRegistryGetVersionUnregistered(t *testing.T) {
	registry := NewRegistry()
	if got := registry.GetVersion(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, "1.0.0"); got != nil {
		t.Fatalf("expected nil for unregistered system, got %v", got)
	}
}

func TestRegistryGetVersionNoDefault(t *testing.T) {
	registry := NewRegistry()
	// No systems registered, empty version should return nil
	if got := registry.GetVersion(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, ""); got != nil {
		t.Fatalf("expected nil for no-default system, got %v", got)
	}
}

func TestRegistryMustGet(t *testing.T) {
	registry := NewRegistry()
	primary := &testSystem{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: "1.0.0"}
	registry.Register(primary)

	if got := registry.MustGet(primary.ID()); got != primary {
		t.Fatalf("MustGet = %v, want primary", got)
	}
}

func TestRegistryMustGetPanics(t *testing.T) {
	registry := NewRegistry()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unregistered system")
		}
	}()
	registry.MustGet(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()
	primary := &testSystem{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: "1.0.0"}
	registry.Register(primary)

	systems := registry.List()
	if len(systems) != 1 {
		t.Fatalf("List() returned %d systems, want 1", len(systems))
	}
	if systems[0] != primary {
		t.Fatalf("List()[0] = %v, want primary", systems[0])
	}
}

func TestRegistryListEmpty(t *testing.T) {
	registry := NewRegistry()
	systems := registry.List()
	if len(systems) != 0 {
		t.Fatalf("List() returned %d systems, want 0", len(systems))
	}
}
