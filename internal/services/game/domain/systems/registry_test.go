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
	if got := registry.GetVersion(primary.ID(), "1.1.0"); got != secondary {
		t.Fatalf("GetVersion = %v, want secondary", got)
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
