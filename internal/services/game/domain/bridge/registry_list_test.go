package bridge

import (
	"testing"
)

func TestAdapterRegistryAdapters(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *AdapterRegistry
		if got := registry.Adapters(); got != nil {
			t.Fatalf("Adapters() = %v, want nil", got)
		}
	})

	t.Run("returns snapshot copy", func(t *testing.T) {
		registry := NewAdapterRegistry()
		adapterA := &testAdapter{id: "alpha", version: "1.0.0"}
		adapterB := &testAdapter{id: "beta", version: "2.0.0"}
		if err := registry.Register(adapterA); err != nil {
			t.Fatalf("register adapterA: %v", err)
		}
		if err := registry.Register(adapterB); err != nil {
			t.Fatalf("register adapterB: %v", err)
		}

		got := registry.Adapters()
		if len(got) != 2 {
			t.Fatalf("Adapters() len = %d, want 2", len(got))
		}
		seen := map[*testAdapter]bool{}
		for _, adapter := range got {
			if cast, ok := adapter.(*testAdapter); ok {
				seen[cast] = true
			}
		}
		if !seen[adapterA] || !seen[adapterB] {
			t.Fatalf("Adapters() missing registered adapters: %+v", got)
		}

		// Mutate caller slice and ensure registry storage is unaffected.
		got[0] = nil
		fresh := registry.Adapters()
		if len(fresh) != 2 {
			t.Fatalf("fresh Adapters() len = %d, want 2", len(fresh))
		}
	})
}

func TestMetadataRegistryDefaultVersion(t *testing.T) {
	registry := NewMetadataRegistry()
	systemID := SystemIDDaggerheart

	if got := registry.DefaultVersion(systemID); got != "" {
		t.Fatalf("DefaultVersion() = %q, want empty", got)
	}

	first := &testGameSystem{id: systemID, version: "1.0.0"}
	second := &testGameSystem{id: systemID, version: "2.0.0"}
	if err := registry.Register(first); err != nil {
		t.Fatalf("register first: %v", err)
	}
	if err := registry.Register(second); err != nil {
		t.Fatalf("register second: %v", err)
	}

	if got := registry.DefaultVersion(systemID); got != "1.0.0" {
		t.Fatalf("DefaultVersion() = %q, want %q", got, "1.0.0")
	}
}

func TestMetadataRegistryList(t *testing.T) {
	registry := NewMetadataRegistry()
	alpha := &testGameSystem{id: SystemIDDaggerheart, version: "1.0.0"}
	beta := &testGameSystem{id: SystemIDDaggerheart, version: "2.0.0"}
	if err := registry.Register(alpha); err != nil {
		t.Fatalf("register alpha: %v", err)
	}
	if err := registry.Register(beta); err != nil {
		t.Fatalf("register beta: %v", err)
	}

	listed := registry.List()
	if len(listed) != 2 {
		t.Fatalf("List() len = %d, want 2", len(listed))
	}

	seen := map[string]bool{}
	for _, system := range listed {
		seen[system.Version()] = true
	}
	if !seen[alpha.version] || !seen[beta.version] {
		t.Fatalf("List() missing registered systems: %+v", listed)
	}

	// Mutate caller slice and ensure registry state is unaffected.
	listed[0] = nil
	fresh := registry.List()
	if len(fresh) != 2 {
		t.Fatalf("fresh List() len = %d, want 2", len(fresh))
	}
}
