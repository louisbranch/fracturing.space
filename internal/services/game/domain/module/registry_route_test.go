package module

import (
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestResolveModule(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
		t.Fatalf("register module: %v", err)
	}

	t.Run("nil registry", func(t *testing.T) {
		_, _, _, err := resolveModule(nil, "daggerheart", "v1")
		if !errors.Is(err, ErrRegistryRequired) {
			t.Fatalf("expected ErrRegistryRequired, got %v", err)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		_, _, _, err := resolveModule(registry, " ", "v1")
		if !errors.Is(err, ErrSystemIDRequired) {
			t.Fatalf("expected ErrSystemIDRequired, got %v", err)
		}
	})

	t.Run("missing version", func(t *testing.T) {
		_, _, _, err := resolveModule(registry, "daggerheart", " ")
		if !errors.Is(err, ErrSystemVersionRequired) {
			t.Fatalf("expected ErrSystemVersionRequired, got %v", err)
		}
	})

	t.Run("missing module", func(t *testing.T) {
		_, id, version, err := resolveModule(registry, "missing", "v2")
		if !errors.Is(err, ErrModuleNotFound) {
			t.Fatalf("expected ErrModuleNotFound, got %v", err)
		}
		if id != "missing" || version != "v2" {
			t.Fatalf("resolved coordinates = %s@%s, want missing@v2", id, version)
		}
	})

	t.Run("resolves trimmed coordinates", func(t *testing.T) {
		module, id, version, err := resolveModule(registry, " daggerheart ", " v1 ")
		if err != nil {
			t.Fatalf("resolve module: %v", err)
		}
		if module == nil {
			t.Fatal("expected module")
		}
		if id != "daggerheart" || version != "v1" {
			t.Fatalf("resolved coordinates = %s@%s, want daggerheart@v1", id, version)
		}
	})
}

func TestRouteCommand_RegistryAndVersionGuards(t *testing.T) {
	cmd := command.Command{
		Type:          command.Type("system.test"),
		SystemID:      "daggerheart",
		SystemVersion: "v1",
	}

	t.Run("nil registry", func(t *testing.T) {
		_, err := RouteCommand(nil, nil, cmd, nil)
		if !errors.Is(err, ErrRegistryRequired) {
			t.Fatalf("expected ErrRegistryRequired, got %v", err)
		}
	})

	t.Run("missing version", func(t *testing.T) {
		registry := NewRegistry()
		_, err := RouteCommand(registry, nil, command.Command{
			Type:     command.Type("system.test"),
			SystemID: "daggerheart",
		}, nil)
		if !errors.Is(err, ErrSystemVersionRequired) {
			t.Fatalf("expected ErrSystemVersionRequired, got %v", err)
		}
	})
}

func TestRouteEvent_RegistryAndIdentifierGuards(t *testing.T) {
	evt := event.Event{
		Type:          event.Type("system.event"),
		SystemID:      "daggerheart",
		SystemVersion: "v1",
	}

	t.Run("nil registry", func(t *testing.T) {
		_, err := RouteEvent(nil, nil, evt)
		if !errors.Is(err, ErrRegistryRequired) {
			t.Fatalf("expected ErrRegistryRequired, got %v", err)
		}
	})

	t.Run("missing id", func(t *testing.T) {
		registry := NewRegistry()
		_, err := RouteEvent(registry, nil, event.Event{
			Type:          event.Type("system.event"),
			SystemVersion: "v1",
		})
		if !errors.Is(err, ErrSystemIDRequired) {
			t.Fatalf("expected ErrSystemIDRequired, got %v", err)
		}
	})

	t.Run("missing version", func(t *testing.T) {
		registry := NewRegistry()
		_, err := RouteEvent(registry, nil, event.Event{
			Type:     event.Type("system.event"),
			SystemID: "daggerheart",
		})
		if !errors.Is(err, ErrSystemVersionRequired) {
			t.Fatalf("expected ErrSystemVersionRequired, got %v", err)
		}
	})
}

func TestRegistryRegister_Branches(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var registry *Registry
		err := registry.Register(stubModule{id: "daggerheart", version: "v1"})
		if !errors.Is(err, ErrRegistryRequired) {
			t.Fatalf("expected ErrRegistryRequired, got %v", err)
		}
	})

	t.Run("nil module", func(t *testing.T) {
		registry := NewRegistry()
		err := registry.Register(nil)
		if err == nil {
			t.Fatal("expected error for nil module")
		}
		if err.Error() != "system module is required" {
			t.Fatalf("error = %q, want %q", err.Error(), "system module is required")
		}
	})

	t.Run("initializes maps for zero-value registry", func(t *testing.T) {
		registry := &Registry{}
		if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
			t.Fatalf("register module: %v", err)
		}
		if registry.modules == nil {
			t.Fatal("expected modules map to be initialized")
		}
		if registry.defaults == nil {
			t.Fatal("expected defaults map to be initialized")
		}
		if got := registry.defaults["daggerheart"]; got != "v1" {
			t.Fatalf("default version = %q, want %q", got, "v1")
		}
	})

	t.Run("duplicate registration", func(t *testing.T) {
		registry := NewRegistry()
		module := stubModule{id: "daggerheart", version: "v1"}
		if err := registry.Register(module); err != nil {
			t.Fatalf("register first module: %v", err)
		}
		err := registry.Register(module)
		if !errors.Is(err, ErrSystemAlreadyRegistered) {
			t.Fatalf("expected ErrSystemAlreadyRegistered, got %v", err)
		}
	})
}

func TestRegistryGet_Branches(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var registry *Registry
		if got := registry.Get("daggerheart", "v1"); got != nil {
			t.Fatalf("Get() = %v, want nil", got)
		}
	})

	t.Run("blank id", func(t *testing.T) {
		registry := NewRegistry()
		if got := registry.Get("  ", "v1"); got != nil {
			t.Fatalf("Get() = %v, want nil", got)
		}
	})

	t.Run("missing default version", func(t *testing.T) {
		registry := NewRegistry()
		if got := registry.Get("daggerheart", ""); got != nil {
			t.Fatalf("Get() = %v, want nil", got)
		}
	})

	t.Run("trimmed explicit version", func(t *testing.T) {
		registry := NewRegistry()
		if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
			t.Fatalf("register module: %v", err)
		}
		module := registry.Get(" daggerheart ", " v1 ")
		if module == nil {
			t.Fatal("expected module")
		}
		if module.Version() != "v1" {
			t.Fatalf("version = %q, want %q", module.Version(), "v1")
		}
	})
}
