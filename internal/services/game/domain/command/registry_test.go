package command

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRegistryValidateForDecision_MissingCampaignID(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.create"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	cmd := Command{
		Type:        Type("campaign.create"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{}"),
	}

	_, err := registry.ValidateForDecision(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrCampaignIDRequired) {
		t.Fatalf("expected ErrCampaignIDRequired, got %v", err)
	}
}

func TestRegistryValidateForDecision_UnknownType(t *testing.T) {
	registry := NewRegistry()

	cmd := Command{
		CampaignID:  "camp-1",
		Type:        Type("unknown.command"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{}"),
	}

	_, err := registry.ValidateForDecision(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrTypeUnknown) {
		t.Fatalf("expected ErrTypeUnknown, got %v", err)
	}
}

func TestRegistryValidateForDecision_MissingType(t *testing.T) {
	registry := NewRegistry()

	cmd := Command{
		CampaignID:  "camp-1",
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{}"),
	}

	_, err := registry.ValidateForDecision(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrTypeRequired) {
		t.Fatalf("expected ErrTypeRequired, got %v", err)
	}
}

func TestRegistryValidateForDecision_SystemCommandRequiresSystemMetadata(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("action.system_test"),
		Owner: OwnerSystem,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	cmd := Command{
		CampaignID:  "camp-1",
		Type:        Type("action.system_test"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{}"),
	}

	_, err := registry.ValidateForDecision(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrSystemMetadataRequired) {
		t.Fatalf("expected ErrSystemMetadataRequired, got %v", err)
	}
}

func TestRegistryValidateForDecision_SystemCommandTypeMustMatchSystemID(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("sys.alpha.action.test"),
		Owner: OwnerSystem,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	cmd := Command{
		CampaignID:    "camp-1",
		Type:          Type("sys.alpha.action.test"),
		ActorType:     ActorTypeSystem,
		SystemID:      "beta",
		SystemVersion: "v1",
		PayloadJSON:   []byte("{}"),
	}

	_, err := registry.ValidateForDecision(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "system id must match command type namespace") {
		t.Fatalf("expected system-id namespace mismatch error, got %v", err)
	}
}

func TestRegistryValidateForDecision_CoreCommandRejectsSystemMetadata(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.create"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	cmd := Command{
		CampaignID:    "camp-1",
		Type:          Type("campaign.create"),
		ActorType:     ActorTypeSystem,
		SystemID:      "sys-1",
		SystemVersion: "1.0",
		PayloadJSON:   []byte("{}"),
	}

	_, err := registry.ValidateForDecision(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrSystemMetadataForbidden) {
		t.Fatalf("expected ErrSystemMetadataForbidden, got %v", err)
	}
}

func TestRegistryValidateForDecision_InvalidActorType(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.create"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	cmd := Command{
		CampaignID:  "camp-1",
		Type:        Type("campaign.create"),
		ActorType:   ActorType("alien"),
		PayloadJSON: []byte("{}"),
	}

	_, err := registry.ValidateForDecision(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrActorTypeInvalid) {
		t.Fatalf("expected ErrActorTypeInvalid, got %v", err)
	}
}

func TestRegistryValidateForDecision_MissingActorID(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.create"),
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
			cmd := Command{
				CampaignID:  "camp-1",
				Type:        Type("campaign.create"),
				ActorType:   tt.actorType,
				PayloadJSON: []byte("{}"),
			}

			_, err := registry.ValidateForDecision(cmd)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrActorIDRequired) {
				t.Fatalf("expected ErrActorIDRequired, got %v", err)
			}
		})
	}
}

func TestRegistryValidateForDecision_InvalidPayloadJSON(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.create"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	cmd := Command{
		CampaignID:  "camp-1",
		Type:        Type("campaign.create"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{"),
	}

	_, err := registry.ValidateForDecision(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrPayloadInvalid) {
		t.Fatalf("expected ErrPayloadInvalid, got %v", err)
	}
}

func TestRegistryValidateForDecision_CanonicalizesPayloadJSON(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.create"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	cmd := Command{
		CampaignID:  "camp-1",
		Type:        Type("campaign.create"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{\"b\":2,\"a\":1}"),
	}

	normalized, err := registry.ValidateForDecision(cmd)
	if err != nil {
		t.Fatalf("validate command: %v", err)
	}
	if string(normalized.PayloadJSON) != `{"a":1,"b":2}` {
		t.Fatalf("PayloadJSON = %s, want %s", string(normalized.PayloadJSON), `{"a":1,"b":2}`)
	}
}

func TestRegistryValidateForDecision_PayloadValidatorUsesCanonicalJSON(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("campaign.create"),
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

	cmd := Command{
		CampaignID:  "camp-1",
		Type:        Type("campaign.create"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{\"b\":2,\"a\":1}"),
	}

	_, err := registry.ValidateForDecision(cmd)
	if err != nil {
		t.Fatalf("validate command: %v", err)
	}
}

func TestRegistryRegister_RequiresType(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register(Definition{Owner: OwnerCore})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrTypeRequired) {
		t.Fatalf("expected ErrTypeRequired, got %v", err)
	}
}

func TestRegistryRegister_InvalidOwner(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register(Definition{Type: Type("campaign.create"), Owner: Owner("unknown")})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "owner must be core or system") {
		t.Fatalf("expected owner validation error, got %v", err)
	}
}

func TestRegistryRegister_DuplicateType(t *testing.T) {
	registry := NewRegistry()
	def := Definition{Type: Type("campaign.create"), Owner: OwnerCore}
	if err := registry.Register(def); err != nil {
		t.Fatalf("register first: %v", err)
	}

	err := registry.Register(def)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "command type already registered") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}
