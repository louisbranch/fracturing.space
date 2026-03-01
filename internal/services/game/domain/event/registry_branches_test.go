package event

import (
	"errors"
	"testing"
)

func TestRegistryRegister_BranchCoverage(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *Registry
		err := registry.Register(Definition{Type: Type("campaign.created"), Owner: OwnerCore})
		if err == nil || err.Error() != "registry is required" {
			t.Fatalf("Register() error = %v, want registry is required", err)
		}
	})

	t.Run("invalid owner", func(t *testing.T) {
		registry := NewRegistry()
		err := registry.Register(Definition{Type: Type("campaign.created"), Owner: Owner("unknown")})
		if err == nil || err.Error() != "owner must be core or system" {
			t.Fatalf("Register() error = %v, want owner validation", err)
		}
	})

	t.Run("invalid addressing", func(t *testing.T) {
		registry := NewRegistry()
		err := registry.Register(Definition{
			Type:       Type("campaign.created"),
			Owner:      OwnerCore,
			Addressing: AddressingPolicy(99),
		})
		if err == nil || err.Error() != "event addressing policy is invalid" {
			t.Fatalf("Register() error = %v, want addressing validation", err)
		}
	})

	t.Run("duplicate type", func(t *testing.T) {
		registry := NewRegistry()
		first := Definition{Type: Type("campaign.created"), Owner: OwnerCore}
		if err := registry.Register(first); err != nil {
			t.Fatalf("register first: %v", err)
		}
		err := registry.Register(first)
		if err == nil || err.Error() != "event type already registered: campaign.created" {
			t.Fatalf("Register() error = %v, want duplicate type rejection", err)
		}
	})
}

func TestRegistryResolve_BranchCoverage(t *testing.T) {
	t.Run("nil registry returns input", func(t *testing.T) {
		var registry *Registry
		if got := registry.Resolve(Type("legacy")); got != Type("legacy") {
			t.Fatalf("Resolve() = %s, want legacy", got)
		}
	})

	t.Run("no alias map returns input", func(t *testing.T) {
		registry := NewRegistry()
		if got := registry.Resolve(Type("legacy")); got != Type("legacy") {
			t.Fatalf("Resolve() = %s, want legacy", got)
		}
	})

	t.Run("registered alias resolves", func(t *testing.T) {
		registry := NewRegistry()
		if err := registry.Register(Definition{Type: Type("participant.updated"), Owner: OwnerCore}); err != nil {
			t.Fatalf("register canonical: %v", err)
		}
		if err := registry.RegisterAlias(Type("participant.changed"), Type("participant.updated")); err != nil {
			t.Fatalf("register alias: %v", err)
		}
		if got := registry.Resolve(Type("participant.changed")); got != Type("participant.updated") {
			t.Fatalf("Resolve() = %s, want participant.updated", got)
		}
	})
}

func TestRegistryValidateForAppend_StorageFieldsSet(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{Type: Type("campaign.created"), Owner: OwnerCore}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	_, err := registry.ValidateForAppend(Event{
		CampaignID: "camp-1",
		Seq:        1,
		Type:       Type("campaign.created"),
	})
	if !errors.Is(err, ErrStorageFieldsSet) {
		t.Fatalf("expected ErrStorageFieldsSet, got %v", err)
	}
}
