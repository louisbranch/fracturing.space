package command

import "testing"

func TestRegistryDefinition(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *Registry
		if def, ok := registry.Definition(Type("campaign.create")); ok || def.Type != "" || def.Owner != "" {
			t.Fatalf("Definition() = (%+v, %v), want empty definition and false", def, ok)
		}
	})

	t.Run("blank type", func(t *testing.T) {
		registry := NewRegistry()
		if def, ok := registry.Definition(Type(" ")); ok || def.Type != "" || def.Owner != "" {
			t.Fatalf("Definition() = (%+v, %v), want empty definition and false", def, ok)
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		registry := NewRegistry()
		if def, ok := registry.Definition(Type("campaign.create")); ok || def.Type != "" || def.Owner != "" {
			t.Fatalf("Definition() = (%+v, %v), want empty definition and false", def, ok)
		}
	})

	t.Run("trimmed lookup", func(t *testing.T) {
		registry := NewRegistry()
		expected := Definition{
			Type:  Type("campaign.create"),
			Owner: OwnerCore,
			Gate: GatePolicy{
				Scope:         GateScopeSession,
				AllowWhenOpen: true,
			},
		}
		if err := registry.Register(expected); err != nil {
			t.Fatalf("register definition: %v", err)
		}

		got, ok := registry.Definition(Type("  campaign.create  "))
		if !ok {
			t.Fatal("Definition() ok = false, want true")
		}
		if got.Type != expected.Type || got.Owner != expected.Owner || got.Gate != expected.Gate {
			t.Fatalf("Definition() = %+v, want %+v", got, expected)
		}
	})
}
