package event

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRegistryRegister_RejectsBlankTypeAfterTrim(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register(Definition{Type: Type("   "), Owner: OwnerCore})
	if !errors.Is(err, ErrTypeRequired) {
		t.Fatalf("Register() error = %v, want %v", err, ErrTypeRequired)
	}
}

func TestRegistryRegister_InitializesDefinitionsMapWhenNil(t *testing.T) {
	registry := &Registry{}
	if err := registry.Register(Definition{Type: Type("core.event"), Owner: OwnerCore}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, ok := registry.definitions[Type("core.event")]; !ok {
		t.Fatal("expected definitions map to contain registered type")
	}
}

func TestRegistryValidateForAppend_NilRegistry(t *testing.T) {
	var registry *Registry
	_, err := registry.ValidateForAppend(Event{})
	if err == nil || err.Error() != "registry is required" {
		t.Fatalf("ValidateForAppend() error = %v, want registry is required", err)
	}
}

func TestRegistryValidateForAppend_RequiresCampaignID(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{Type: Type("core.event"), Owner: OwnerCore}); err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err := registry.ValidateForAppend(Event{
		Type:        Type("core.event"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte(`{}`),
	})
	if !errors.Is(err, ErrCampaignIDRequired) {
		t.Fatalf("ValidateForAppend() error = %v, want %v", err, ErrCampaignIDRequired)
	}
}

func TestRegistryValidateForAppend_RequiresType(t *testing.T) {
	registry := NewRegistry()
	_, err := registry.ValidateForAppend(Event{
		CampaignID:  "camp-1",
		Type:        Type(" "),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte(`{}`),
	})
	if !errors.Is(err, ErrTypeRequired) {
		t.Fatalf("ValidateForAppend() error = %v, want %v", err, ErrTypeRequired)
	}
}

func TestRegistryValidateForAppend_DefaultsActorTypeToSystem(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{Type: Type("core.event"), Owner: OwnerCore}); err != nil {
		t.Fatalf("register: %v", err)
	}

	normalized, err := registry.ValidateForAppend(Event{
		CampaignID:  "camp-1",
		Type:        Type("core.event"),
		Timestamp:   time.Unix(0, 0).UTC(),
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("ValidateForAppend() error = %v", err)
	}
	if normalized.ActorType != ActorTypeSystem {
		t.Fatalf("ActorType = %s, want %s", normalized.ActorType, ActorTypeSystem)
	}
}

func TestRegistryValidateForAppend_AddressingPolicyEntityType(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:       Type("core.entity_type_only"),
		Owner:      OwnerCore,
		Addressing: AddressingPolicyEntityType,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	normalized, err := registry.ValidateForAppend(Event{
		CampaignID:  "camp-1",
		Type:        Type("core.entity_type_only"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		EntityType:  "campaign",
		PayloadJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("ValidateForAppend() error = %v", err)
	}
	if normalized.EntityType != "campaign" || normalized.EntityID != "" {
		t.Fatalf("normalized entity = (%s,%s), want (campaign,empty)", normalized.EntityType, normalized.EntityID)
	}
}

func TestRegistryValidateForAppend_RejectsEntityIDWithoutType(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{Type: Type("core.event"), Owner: OwnerCore}); err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err := registry.ValidateForAppend(Event{
		CampaignID:  "camp-1",
		Type:        Type("core.event"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		EntityID:    "entity-1",
		PayloadJSON: []byte(`{}`),
	})
	if !errors.Is(err, ErrEntityTypeRequired) {
		t.Fatalf("ValidateForAppend() error = %v, want %v", err, ErrEntityTypeRequired)
	}
}

func TestRegistryValidateForAppend_RejectsInvalidDefinitionOwner(t *testing.T) {
	registry := &Registry{
		definitions: map[Type]Definition{
			Type("core.invalid_owner"): {
				Type:  Type("core.invalid_owner"),
				Owner: Owner("invalid"),
			},
		},
	}
	_, err := registry.ValidateForAppend(Event{
		CampaignID:  "camp-1",
		Type:        Type("core.invalid_owner"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte(`{}`),
	})
	if err == nil || err.Error() != "event owner is invalid" {
		t.Fatalf("ValidateForAppend() error = %v, want event owner is invalid", err)
	}
}

func TestRegistryValidateForAppend_CanonicalPayloadFailure(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{Type: Type("core.event"), Owner: OwnerCore}); err != nil {
		t.Fatalf("register: %v", err)
	}

	restore := canonicalJSON
	canonicalJSON = func(any) ([]byte, error) {
		return nil, errors.New("forced canonical failure")
	}
	t.Cleanup(func() { canonicalJSON = restore })

	_, err := registry.ValidateForAppend(Event{
		CampaignID:  "camp-1",
		Type:        Type("core.event"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte(`{"ok":true}`),
	})
	if err == nil {
		t.Fatal("expected canonical payload failure")
	}
	if !strings.Contains(err.Error(), "canonical payload json") {
		t.Fatalf("ValidateForAppend() error = %v, want canonical payload context", err)
	}
}

func TestRegistryValidateForAppend_WrapsPayloadValidatorError(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("core.event"),
		Owner: OwnerCore,
		ValidatePayload: func(json.RawMessage) error {
			return errors.New("validator failed")
		},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err := registry.ValidateForAppend(Event{
		CampaignID:  "camp-1",
		Type:        Type("core.event"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte(`{"ok":true}`),
	})
	if err == nil {
		t.Fatal("expected payload validator error")
	}
	if !strings.Contains(err.Error(), "payload invalid") {
		t.Fatalf("ValidateForAppend() error = %v, want payload invalid prefix", err)
	}
}

func TestRegistryRegisterAlias_Branches(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *Registry
		err := registry.RegisterAlias(Type("legacy"), Type("canonical"))
		if err == nil || err.Error() != "registry is required" {
			t.Fatalf("RegisterAlias() error = %v, want registry is required", err)
		}
	})

	t.Run("blank types", func(t *testing.T) {
		registry := NewRegistry()
		if err := registry.Register(Definition{Type: Type("core.event"), Owner: OwnerCore}); err != nil {
			t.Fatalf("register: %v", err)
		}
		err := registry.RegisterAlias(Type(" "), Type("core.event"))
		if err == nil || err.Error() != "deprecated and canonical types are required" {
			t.Fatalf("RegisterAlias() error = %v, want required types error", err)
		}
	})
}

func TestRegistryResolve_ReturnsInputWhenAliasMissingInNonEmptyMap(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{Type: Type("core.canonical"), Owner: OwnerCore}); err != nil {
		t.Fatalf("register canonical: %v", err)
	}
	if err := registry.RegisterAlias(Type("core.legacy"), Type("core.canonical")); err != nil {
		t.Fatalf("register alias: %v", err)
	}
	if got := registry.Resolve(Type("core.other")); got != Type("core.other") {
		t.Fatalf("Resolve() = %s, want core.other", got)
	}
}

func TestRegistryMissingPayloadValidators_NilAndEmptyRegistry(t *testing.T) {
	t.Run("nil registry", func(t *testing.T) {
		var registry *Registry
		if got := registry.MissingPayloadValidators(); got != nil {
			t.Fatalf("MissingPayloadValidators() = %v, want nil", got)
		}
	})

	t.Run("empty registry", func(t *testing.T) {
		registry := NewRegistry()
		if got := registry.MissingPayloadValidators(); got != nil {
			t.Fatalf("MissingPayloadValidators() = %v, want nil", got)
		}
	})
}
