package module

import "testing"

func TestRegistryDefaultVersion(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *Registry
		if got := registry.DefaultVersion("daggerheart"); got != "" {
			t.Fatalf("DefaultVersion() = %q, want empty", got)
		}
	})

	t.Run("missing default", func(t *testing.T) {
		registry := NewRegistry()
		if got := registry.DefaultVersion("daggerheart"); got != "" {
			t.Fatalf("DefaultVersion() = %q, want empty", got)
		}
	})

	t.Run("trimmed lookup", func(t *testing.T) {
		registry := NewRegistry()
		if err := registry.Register(stubModule{id: "daggerheart", version: "v1"}); err != nil {
			t.Fatalf("register module: %v", err)
		}
		if got := registry.DefaultVersion(" daggerheart "); got != "v1" {
			t.Fatalf("DefaultVersion() = %q, want %q", got, "v1")
		}
	})
}

func TestRegistryList(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *Registry
		if got := registry.List(); got != nil {
			t.Fatalf("List() = %v, want nil", got)
		}
	})

	t.Run("empty registry", func(t *testing.T) {
		registry := NewRegistry()
		got := registry.List()
		if got == nil || len(got) != 0 {
			t.Fatalf("List() = %v, want empty non-nil slice", got)
		}
	})

	t.Run("returns snapshot copy", func(t *testing.T) {
		registry := NewRegistry()
		first := stubModule{id: "daggerheart", version: "v1"}
		second := stubModule{id: "daggerheart", version: "legacy"}
		if err := registry.Register(first); err != nil {
			t.Fatalf("register first: %v", err)
		}
		if err := registry.Register(second); err != nil {
			t.Fatalf("register second: %v", err)
		}

		listed := registry.List()
		if len(listed) != 2 {
			t.Fatalf("List() len = %d, want 2", len(listed))
		}

		seen := map[string]bool{}
		for _, module := range listed {
			seen[module.Version()] = true
		}
		if !seen["v1"] || !seen["legacy"] {
			t.Fatalf("List() missing modules: %+v", listed)
		}

		listed[0] = nil
		fresh := registry.List()
		if len(fresh) != 2 {
			t.Fatalf("fresh List() len = %d, want 2", len(fresh))
		}
	})
}
