package event

import (
	"encoding/json"
	"errors"
	"strings"
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

func TestRegistryValidateForAppend_SystemEventTypeMustMatchSystemID(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("sys.alpha.action.tested"),
		Owner: OwnerSystem,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	evt := Event{
		CampaignID:    "camp-1",
		Type:          Type("sys.alpha.action.tested"),
		Timestamp:     time.Unix(0, 0).UTC(),
		ActorType:     ActorTypeSystem,
		SystemID:      "beta",
		SystemVersion: "v1",
		EntityType:    "action",
		EntityID:      "req-1",
		PayloadJSON:   []byte("{}"),
	}

	_, err := registry.ValidateForAppend(evt)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "system id must match event type namespace") {
		t.Fatalf("expected system-id namespace mismatch error, got %v", err)
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

func TestRegistryRegister_AcceptsIntentReplayOnly(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register(Definition{
		Type:   Type("action.test_replay"),
		Owner:  OwnerCore,
		Intent: IntentReplayOnly,
	})
	if err != nil {
		t.Fatalf("expected IntentReplayOnly to be accepted, got: %v", err)
	}
	def, ok := registry.Definition(Type("action.test_replay"))
	if !ok {
		t.Fatal("expected definition to be found")
	}
	if def.Intent != IntentReplayOnly {
		t.Fatalf("intent = %s, want %s", def.Intent, IntentReplayOnly)
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

func TestRegistryValidateForAppend_RejectsZeroTimestamp(t *testing.T) {
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
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte("{}"),
	}

	_, err := registry.ValidateForAppend(evt)
	if err == nil {
		t.Fatal("expected error for zero timestamp")
	}
	if !errors.Is(err, ErrTimestampRequired) {
		t.Fatalf("expected ErrTimestampRequired, got %v", err)
	}
}

func TestRegistryRegisterAlias_ResolvesDeprecatedToCanonical(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("participant.seat_reassigned"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}
	if err := registry.RegisterAlias(Type("seat.reassigned"), Type("participant.seat_reassigned")); err != nil {
		t.Fatalf("register alias: %v", err)
	}

	got := registry.Resolve(Type("seat.reassigned"))
	if got != Type("participant.seat_reassigned") {
		t.Fatalf("Resolve(seat.reassigned) = %s, want participant.seat_reassigned", got)
	}
}

func TestRegistryResolve_ReturnsInputWhenNoAlias(t *testing.T) {
	registry := NewRegistry()
	got := registry.Resolve(Type("campaign.created"))
	if got != Type("campaign.created") {
		t.Fatalf("Resolve(campaign.created) = %s, want campaign.created", got)
	}
}

func TestRegistryRegisterAlias_RejectsUnknownCanonical(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterAlias(Type("old.type"), Type("new.type"))
	if err == nil {
		t.Fatal("expected error for unknown canonical type")
	}
}

func TestRegistryRegisterAlias_RejectsDuplicateAlias(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("participant.seat_reassigned"),
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}
	if err := registry.RegisterAlias(Type("seat.reassigned"), Type("participant.seat_reassigned")); err != nil {
		t.Fatalf("register first alias: %v", err)
	}
	err := registry.RegisterAlias(Type("seat.reassigned"), Type("participant.seat_reassigned"))
	if err == nil {
		t.Fatal("expected error for duplicate alias")
	}
}

func TestRegistryMissingPayloadValidators_ReportsNonAuditWithoutValidator(t *testing.T) {
	registry := NewRegistry()
	// Non-audit event without validator — should be reported.
	if err := registry.Register(Definition{
		Type:  "ev.no_validator",
		Owner: OwnerCore,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Audit event without validator — should NOT be reported.
	if err := registry.Register(Definition{
		Type:   "ev.audit",
		Owner:  OwnerCore,
		Intent: IntentAuditOnly,
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Non-audit event WITH validator — should NOT be reported.
	if err := registry.Register(Definition{
		Type:            "ev.has_validator",
		Owner:           OwnerCore,
		ValidatePayload: func(json.RawMessage) error { return nil },
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	missing := registry.MissingPayloadValidators()
	if len(missing) != 1 {
		t.Fatalf("MissingPayloadValidators() returned %d types, want 1: %v", len(missing), missing)
	}
	if missing[0] != "ev.no_validator" {
		t.Fatalf("missing[0] = %s, want ev.no_validator", missing[0])
	}
}

func TestRegistryMissingPayloadValidators_EmptyForFullCoverage(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:            "ev.validated",
		Owner:           OwnerCore,
		ValidatePayload: func(json.RawMessage) error { return nil },
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	missing := registry.MissingPayloadValidators()
	if len(missing) != 0 {
		t.Fatalf("MissingPayloadValidators() returned %d types, want 0", len(missing))
	}
}

func TestRegistryShouldFold(t *testing.T) {
	registry := NewRegistry()
	for _, def := range []Definition{
		{Type: "ev.projected", Owner: OwnerCore, Intent: IntentProjectionAndReplay},
		{Type: "ev.replay", Owner: OwnerCore, Intent: IntentReplayOnly},
		{Type: "ev.audit", Owner: OwnerCore, Intent: IntentAuditOnly},
	} {
		if err := registry.Register(def); err != nil {
			t.Fatalf("register %s: %v", def.Type, err)
		}
	}

	tests := []struct {
		eventType Type
		want      bool
	}{
		{"ev.projected", true},
		{"ev.replay", true},
		{"ev.audit", false},
		{"ev.unknown", true}, // unknown types default to foldable
	}
	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			if got := registry.ShouldFold(tt.eventType); got != tt.want {
				t.Fatalf("ShouldFold(%s) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestRegistryShouldProject(t *testing.T) {
	registry := NewRegistry()
	for _, def := range []Definition{
		{Type: "ev.projected", Owner: OwnerCore, Intent: IntentProjectionAndReplay},
		{Type: "ev.replay", Owner: OwnerCore, Intent: IntentReplayOnly},
		{Type: "ev.audit", Owner: OwnerCore, Intent: IntentAuditOnly},
	} {
		if err := registry.Register(def); err != nil {
			t.Fatalf("register %s: %v", def.Type, err)
		}
	}

	tests := []struct {
		eventType Type
		want      bool
	}{
		{"ev.projected", true},
		{"ev.replay", false},
		{"ev.audit", false},
		{"ev.unknown", true}, // unknown types default to projectable
	}
	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			if got := registry.ShouldProject(tt.eventType); got != tt.want {
				t.Fatalf("ShouldProject(%s) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}
