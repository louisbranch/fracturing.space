package event

import "testing"

func TestRegistryDefinition(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *Registry
		if def, ok := registry.Definition(Type("campaign.created")); ok || def.Type != "" || def.Owner != "" {
			t.Fatalf("Definition() = (%+v, %v), want empty definition and false", def, ok)
		}
	})

	t.Run("blank type", func(t *testing.T) {
		registry := NewRegistry()
		if def, ok := registry.Definition(Type(" ")); ok || def.Type != "" || def.Owner != "" {
			t.Fatalf("Definition() = (%+v, %v), want empty definition and false", def, ok)
		}
	})

	t.Run("registered type with trim", func(t *testing.T) {
		registry := NewRegistry()
		expected := Definition{
			Type:       Type("campaign.created"),
			Owner:      OwnerCore,
			Addressing: AddressingPolicyEntityTarget,
		}
		if err := registry.Register(expected); err != nil {
			t.Fatalf("register: %v", err)
		}

		got, ok := registry.Definition(Type(" campaign.created "))
		if !ok {
			t.Fatal("Definition() ok = false, want true")
		}
		if got.Type != expected.Type || got.Owner != expected.Owner || got.Addressing != expected.Addressing {
			t.Fatalf("Definition() = %+v, want %+v", got, expected)
		}
	})
}

func TestRegistryListAliases(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *Registry
		if got := registry.ListAliases(); got != nil {
			t.Fatalf("ListAliases() = %v, want nil", got)
		}
	})

	t.Run("empty aliases", func(t *testing.T) {
		registry := NewRegistry()
		if got := registry.ListAliases(); got != nil {
			t.Fatalf("ListAliases() = %v, want nil", got)
		}
	})

	t.Run("returns copy", func(t *testing.T) {
		registry := NewRegistry()
		if err := registry.Register(Definition{Type: Type("campaign.created"), Owner: OwnerCore}); err != nil {
			t.Fatalf("register canonical 1: %v", err)
		}
		if err := registry.Register(Definition{Type: Type("participant.updated"), Owner: OwnerCore}); err != nil {
			t.Fatalf("register canonical 2: %v", err)
		}
		if err := registry.RegisterAlias(Type("legacy.campaign.created"), Type("campaign.created")); err != nil {
			t.Fatalf("register alias 1: %v", err)
		}
		if err := registry.RegisterAlias(Type("participant.changed"), Type("participant.updated")); err != nil {
			t.Fatalf("register alias 2: %v", err)
		}

		got := registry.ListAliases()
		if len(got) != 2 {
			t.Fatalf("ListAliases() len = %d, want 2", len(got))
		}
		if got[Type("legacy.campaign.created")] != Type("campaign.created") {
			t.Fatalf("alias legacy.campaign.created = %s, want campaign.created", got[Type("legacy.campaign.created")])
		}
		if got[Type("participant.changed")] != Type("participant.updated") {
			t.Fatalf("alias participant.changed = %s, want participant.updated", got[Type("participant.changed")])
		}

		// Mutate caller copy and ensure registry copy remains unchanged.
		got[Type("legacy.campaign.created")] = Type("corrupted")
		fresh := registry.ListAliases()
		if fresh[Type("legacy.campaign.created")] != Type("campaign.created") {
			t.Fatalf("fresh alias legacy.campaign.created = %s, want campaign.created", fresh[Type("legacy.campaign.created")])
		}
	})
}
