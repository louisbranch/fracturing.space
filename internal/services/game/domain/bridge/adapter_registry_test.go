package bridge_test

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

// stubAdapter is a minimal bridge.Adapter for testing registry operations.
type stubAdapter struct {
	id      string
	version string
}

func (s *stubAdapter) ID() string                                        { return s.id }
func (s *stubAdapter) Version() string                                   { return s.version }
func (s *stubAdapter) Apply(_ context.Context, _ event.Event) error      { return nil }
func (s *stubAdapter) Snapshot(_ context.Context, _ string) (any, error) { return nil, nil }
func (s *stubAdapter) HandledTypes() []event.Type                        { return nil }

func TestAdapterRegistryHas(t *testing.T) {
	reg := bridge.NewAdapterRegistry()
	adapter := &stubAdapter{id: "test_system", version: "1.0"}
	if err := reg.Register(adapter); err != nil {
		t.Fatalf("register: %v", err)
	}

	t.Run("registered adapter returns true", func(t *testing.T) {
		if !reg.Has("test_system", "1.0") {
			t.Error("Has() = false, want true for registered adapter")
		}
	})

	t.Run("unregistered adapter returns false", func(t *testing.T) {
		if reg.Has("unknown", "1.0") {
			t.Error("Has() = true, want false for unregistered adapter")
		}
	})

	t.Run("nil registry returns false", func(t *testing.T) {
		var nilReg *bridge.AdapterRegistry
		if nilReg.Has("test_system", "1.0") {
			t.Error("Has() = true on nil registry, want false")
		}
	})

	t.Run("empty version uses default", func(t *testing.T) {
		if !reg.Has("test_system", "") {
			t.Error("Has() = false with empty version, want true (default version)")
		}
	})
}

func TestAdapterRegistryGetOptional(t *testing.T) {
	reg := bridge.NewAdapterRegistry()
	adapter := &stubAdapter{id: "test_system", version: "1.0"}
	if err := reg.Register(adapter); err != nil {
		t.Fatalf("register: %v", err)
	}

	t.Run("registered adapter returns adapter and true", func(t *testing.T) {
		got, ok := reg.GetOptional("test_system", "1.0")
		if !ok {
			t.Error("GetOptional() ok = false, want true")
		}
		if got == nil {
			t.Error("GetOptional() adapter = nil, want non-nil")
		}
	})

	t.Run("unregistered adapter returns nil and false", func(t *testing.T) {
		got, ok := reg.GetOptional("unknown", "1.0")
		if ok {
			t.Error("GetOptional() ok = true, want false")
		}
		if got != nil {
			t.Error("GetOptional() adapter = non-nil, want nil")
		}
	})

	t.Run("nil registry returns nil and false", func(t *testing.T) {
		var nilReg *bridge.AdapterRegistry
		got, ok := nilReg.GetOptional("test_system", "1.0")
		if ok {
			t.Error("GetOptional() ok = true on nil registry, want false")
		}
		if got != nil {
			t.Error("GetOptional() adapter = non-nil on nil registry, want nil")
		}
	})

	t.Run("empty version uses default", func(t *testing.T) {
		got, ok := reg.GetOptional("test_system", "")
		if !ok {
			t.Error("GetOptional() ok = false with empty version, want true")
		}
		if got == nil {
			t.Error("GetOptional() adapter = nil with empty version, want non-nil")
		}
	})
}
