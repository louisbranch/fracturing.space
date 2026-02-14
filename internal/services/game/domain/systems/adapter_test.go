package systems

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
)

type testAdapter struct {
	id      commonv1.GameSystem
	version string
}

func (t *testAdapter) ID() commonv1.GameSystem {
	return t.id
}

func (t *testAdapter) Version() string {
	return t.version
}

func (t *testAdapter) ApplyEvent(_ context.Context, _ event.Event) error {
	return nil
}

func (t *testAdapter) Snapshot(_ context.Context, _ string) (any, error) {
	return nil, nil
}

func TestAdapterRegistryDefaultsAndLookup(t *testing.T) {
	registry := NewAdapterRegistry()
	primary := &testAdapter{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: " 1.0.0 "}
	secondary := &testAdapter{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: "1.1.0"}

	registry.Register(primary)
	registry.Register(secondary)

	if got := registry.Get(primary.ID(), ""); got != primary {
		t.Fatalf("Get default = %v, want primary", got)
	}
	if got := registry.Get(primary.ID(), " 1.0.0 "); got != primary {
		t.Fatalf("Get version = %v, want primary", got)
	}
	if got := registry.Get(primary.ID(), "1.1.0"); got != secondary {
		t.Fatalf("Get version = %v, want secondary", got)
	}
}

func TestAdapterRegistryGetNilRegistry(t *testing.T) {
	var registry *AdapterRegistry
	if got := registry.Get(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, "1.0.0"); got != nil {
		t.Fatalf("Get on nil registry = %v, want nil", got)
	}
}

func TestAdapterRegistryRejectsNilRegistry(t *testing.T) {
	var registry *AdapterRegistry
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for nil registry")
		}
	}()

	registry.Register(&testAdapter{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: "1.0.0"})
}

func TestAdapterRegistryRejectsEmptyVersion(t *testing.T) {
	registry := NewAdapterRegistry()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for empty version")
		}
	}()

	registry.Register(&testAdapter{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: " "})
}

func TestAdapterRegistryRejectsDuplicateVersion(t *testing.T) {
	registry := NewAdapterRegistry()
	primary := &testAdapter{id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, version: "1.0.0"}
	registry.Register(primary)

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for duplicate version")
		}
	}()

	registry.Register(primary)
}
