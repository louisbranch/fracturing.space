package event

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestRegistryValidateForAppend_SystemEventRequiresSystemMetadata(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("action.system_test"),
		Owner: OwnerSystem,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	evt := Event{
		CampaignID:  "camp-1",
		Type:        Type("action.system_test"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{}"),
	}

	_, err := registry.ValidateForAppend(evt)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrSystemMetadataRequired) {
		t.Fatalf("expected ErrSystemMetadataRequired, got %v", err)
	}
}

func TestRegistryValidateForAppend_SystemEventRequiresEntityAddressing(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("action.system_test"),
		Owner: OwnerSystem,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	base := Event{
		CampaignID:    "camp-1",
		Type:          Type("action.system_test"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     ActorTypeSystem,
		SystemID:      "sys-1",
		SystemVersion: "1.0.0",
		PayloadJSON:   []byte("{}"),
	}

	_, err := registry.ValidateForAppend(base)
	if err == nil {
		t.Fatal("expected missing entity type error")
	}
	if !errors.Is(err, ErrEntityTypeRequired) {
		t.Fatalf("expected ErrEntityTypeRequired, got %v", err)
	}

	withType := base
	withType.EntityType = "action"
	_, err = registry.ValidateForAppend(withType)
	if err == nil {
		t.Fatal("expected missing entity id error")
	}
	if !errors.Is(err, ErrEntityIDRequired) {
		t.Fatalf("expected ErrEntityIDRequired, got %v", err)
	}

	withTypeAndID := withType
	withTypeAndID.EntityID = "req-1"
	if _, err := registry.ValidateForAppend(withTypeAndID); err != nil {
		t.Fatalf("valid system event rejected: %v", err)
	}
}

func TestRegistryValidateForAppend_DefinitionAddressingPolicy(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:       Type("campaign.created"),
		Owner:      OwnerCore,
		Addressing: AddressingPolicyEntityTarget,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	base := Event{
		CampaignID:  "camp-1",
		Type:        Type("campaign.created"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{}"),
	}

	_, err := registry.ValidateForAppend(base)
	if err == nil {
		t.Fatal("expected missing entity type error")
	}
	if !errors.Is(err, ErrEntityTypeRequired) {
		t.Fatalf("expected ErrEntityTypeRequired, got %v", err)
	}

	withType := base
	withType.EntityType = "campaign"
	_, err = registry.ValidateForAppend(withType)
	if err == nil {
		t.Fatal("expected missing entity id error")
	}
	if !errors.Is(err, ErrEntityIDRequired) {
		t.Fatalf("expected ErrEntityIDRequired, got %v", err)
	}

	withTypeAndID := withType
	withTypeAndID.EntityID = "camp-1"
	if _, err := registry.ValidateForAppend(withTypeAndID); err != nil {
		t.Fatalf("valid addressed event rejected: %v", err)
	}
}

func TestRegistryValidateForAppend_CanonicalizesPayloadJSON(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("action.core_test"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	evt := Event{
		CampaignID:  "camp-1",
		Type:        Type("action.core_test"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{\"b\":2,\"a\":1}"),
	}

	normalized, err := registry.ValidateForAppend(evt)
	if err != nil {
		t.Fatalf("validate event: %v", err)
	}
	if string(normalized.PayloadJSON) != `{"a":1,"b":2}` {
		t.Fatalf("PayloadJSON = %s, want %s", string(normalized.PayloadJSON), `{"a":1,"b":2}`)
	}
}

func TestRegistryValidateForAppend_CoreEventRejectsSystemMetadata(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.created"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	evt := Event{
		CampaignID:    "camp-1",
		Type:          Type("campaign.created"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     ActorTypeSystem,
		SystemID:      "sys-1",
		SystemVersion: "1.0",
		PayloadJSON:   []byte("{}"),
	}

	_, err := registry.ValidateForAppend(evt)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrSystemMetadataForbidden) {
		t.Fatalf("expected ErrSystemMetadataForbidden, got %v", err)
	}
}

func TestRegistryValidateForAppend_UnknownType(t *testing.T) {
	registry := NewRegistry()

	evt := Event{
		CampaignID:  "camp-1",
		Type:        Type("unknown.event"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{}"),
	}

	_, err := registry.ValidateForAppend(evt)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrTypeUnknown) {
		t.Fatalf("expected ErrTypeUnknown, got %v", err)
	}
}

func TestRegistryRegister_DefaultsIntentToProjectionAndReplay(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:       Type("action.test"),
		Owner:      OwnerCore,
		Addressing: AddressingPolicyNone,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}
	definitions := registry.ListDefinitions()
	if len(definitions) != 1 {
		t.Fatalf("definitions length = %d, want 1", len(definitions))
	}
	if definitions[0].Intent != IntentProjectionAndReplay {
		t.Fatalf("intent = %s, want %s", definitions[0].Intent, IntentProjectionAndReplay)
	}
}

func TestRegistryRegister_InvalidIntent(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:   Type("action.test"),
		Owner:  OwnerCore,
		Intent: Intent("invalid-intent"),
	}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistryValidateForAppend_InvalidActorType(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.created"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	evt := Event{
		CampaignID:  "camp-1",
		Type:        Type("campaign.created"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorType("alien"),
		PayloadJSON: []byte("{}"),
	}

	_, err := registry.ValidateForAppend(evt)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrActorTypeInvalid) {
		t.Fatalf("expected ErrActorTypeInvalid, got %v", err)
	}
}

func TestRegistryValidateForAppend_MissingActorID(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.created"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	tests := []struct {
		name      string
		actorType ActorType
	}{
		{name: "participant", actorType: ActorTypeParticipant},
		{name: "gm", actorType: ActorTypeGM},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := Event{
				CampaignID:  "camp-1",
				Type:        Type("campaign.created"),
				Timestamp:   time.Unix(0, 0).UTC(),
				ActorType:   tt.actorType,
				PayloadJSON: []byte("{}"),
			}

			_, err := registry.ValidateForAppend(evt)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrActorIDRequired) {
				t.Fatalf("expected ErrActorIDRequired, got %v", err)
			}
		})
	}
}

func TestRegistryValidateForAppend_InvalidPayloadJSON(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.created"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	evt := Event{
		CampaignID:  "camp-1",
		Type:        Type("campaign.created"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{"),
	}

	_, err := registry.ValidateForAppend(evt)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrPayloadInvalid) {
		t.Fatalf("expected ErrPayloadInvalid, got %v", err)
	}
}

func TestRegistryValidateForAppend_PayloadValidatorUsesCanonicalJSON(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.created"),
		Owner: OwnerCore,
		ValidatePayload: func(raw json.RawMessage) error {
			if string(raw) != `{"a":1,"b":2}` {
				return errors.New("payload not canonical")
			}
			return nil
		},
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	evt := Event{
		CampaignID:  "camp-1",
		Type:        Type("campaign.created"),
		Timestamp:   time.Unix(0, 0).UTC(),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{\"b\":2,\"a\":1}"),
	}

	_, err := registry.ValidateForAppend(evt)
	if err != nil {
		t.Fatalf("validate event: %v", err)
	}
}
