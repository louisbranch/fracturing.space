package command

import "testing"

func TestRegistryListDefinitions(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *Registry
		if got := registry.ListDefinitions(); got != nil {
			t.Fatalf("definitions = %v, want nil", got)
		}
	})

	t.Run("empty registry", func(t *testing.T) {
		registry := NewRegistry()
		if got := registry.ListDefinitions(); got != nil {
			t.Fatalf("definitions = %v, want nil", got)
		}
	})

	t.Run("sorted snapshot", func(t *testing.T) {
		registry := NewRegistry()
		if err := registry.Register(Definition{Type: Type("zeta.run"), Owner: OwnerCore}); err != nil {
			t.Fatalf("register zeta: %v", err)
		}
		if err := registry.Register(Definition{Type: Type("alpha.run"), Owner: OwnerCore}); err != nil {
			t.Fatalf("register alpha: %v", err)
		}

		definitions := registry.ListDefinitions()
		if len(definitions) != 2 {
			t.Fatalf("definitions len = %d, want 2", len(definitions))
		}
		if definitions[0].Type != Type("alpha.run") || definitions[1].Type != Type("zeta.run") {
			t.Fatalf("definition order = [%s, %s], want [alpha.run, zeta.run]", definitions[0].Type, definitions[1].Type)
		}
	})
}
