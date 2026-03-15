package command

import "testing"

func TestTargetEntity(t *testing.T) {
	got := TargetEntity(" participant ", " participant_id ")

	if got.EntityType != "participant" {
		t.Fatalf("EntityType = %q, want %q", got.EntityType, "participant")
	}
	if got.PayloadField != "participant_id" {
		t.Fatalf("PayloadField = %q, want %q", got.PayloadField, "participant_id")
	}
}

func TestRegistryValidateForDecision_NormalizesTargetEntity(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{
		Type:  Type("participant.update"),
		Owner: OwnerCore,
		Target: TargetEntityPolicy{
			EntityType:   "participant",
			PayloadField: "participant_id",
		},
	}); err != nil {
		t.Fatalf("register type: %v", err)
	}

	t.Run("fills EntityType from explicit EntityID", func(t *testing.T) {
		cmd, err := registry.ValidateForDecision(Command{
			CampaignID:  "camp-1",
			Type:        Type("participant.update"),
			ActorType:   ActorTypeSystem,
			EntityID:    " part-1 ",
			PayloadJSON: []byte(`{}`),
		})
		if err != nil {
			t.Fatalf("ValidateForDecision() error = %v", err)
		}
		if cmd.EntityID != "part-1" {
			t.Fatalf("EntityID = %q, want %q", cmd.EntityID, "part-1")
		}
		if cmd.EntityType != "participant" {
			t.Fatalf("EntityType = %q, want %q", cmd.EntityType, "participant")
		}
	})

	t.Run("falls back to payload when EntityID is empty", func(t *testing.T) {
		cmd, err := registry.ValidateForDecision(Command{
			CampaignID:  "camp-1",
			Type:        Type("participant.update"),
			ActorType:   ActorTypeSystem,
			PayloadJSON: []byte(`{"participant_id":" part-2 "}`),
		})
		if err != nil {
			t.Fatalf("ValidateForDecision() error = %v", err)
		}
		if cmd.EntityID != "part-2" {
			t.Fatalf("EntityID = %q, want %q", cmd.EntityID, "part-2")
		}
		if cmd.EntityType != "participant" {
			t.Fatalf("EntityType = %q, want %q", cmd.EntityType, "participant")
		}
	})

	t.Run("explicit EntityID wins over payload fallback", func(t *testing.T) {
		cmd, err := registry.ValidateForDecision(Command{
			CampaignID:  "camp-1",
			Type:        Type("participant.update"),
			ActorType:   ActorTypeSystem,
			EntityID:    "part-3",
			PayloadJSON: []byte(`{"participant_id":"part-4"}`),
		})
		if err != nil {
			t.Fatalf("ValidateForDecision() error = %v", err)
		}
		if cmd.EntityID != "part-3" {
			t.Fatalf("EntityID = %q, want %q", cmd.EntityID, "part-3")
		}
		if cmd.EntityType != "participant" {
			t.Fatalf("EntityType = %q, want %q", cmd.EntityType, "participant")
		}
	})
}

func TestNormalizeTargetEntity(t *testing.T) {
	t.Run("nil command", func(t *testing.T) {
		normalizeTargetEntity(nil, TargetEntity("participant", "participant_id"))
	})

	t.Run("preserves explicit entity type", func(t *testing.T) {
		cmd := Command{
			EntityID:    " part-1 ",
			EntityType:  "custom",
			PayloadJSON: []byte(`{"participant_id":"part-2"}`),
		}

		normalizeTargetEntity(&cmd, TargetEntity("participant", "participant_id"))

		if cmd.EntityID != "part-1" {
			t.Fatalf("EntityID = %q, want %q", cmd.EntityID, "part-1")
		}
		if cmd.EntityType != "custom" {
			t.Fatalf("EntityType = %q, want %q", cmd.EntityType, "custom")
		}
	})
}

func TestPayloadStringField(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		field   string
		want    string
	}{
		{name: "empty field", payload: `{"participant_id":"part-1"}`, field: "", want: ""},
		{name: "missing field", payload: `{"other":"part-1"}`, field: "participant_id", want: ""},
		{name: "malformed payload", payload: `{broken`, field: "participant_id", want: ""},
		{name: "non string field", payload: `{"participant_id":42}`, field: "participant_id", want: ""},
		{name: "trimmed value", payload: `{"participant_id":" part-1 "}`, field: " participant_id ", want: "part-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := payloadStringField([]byte(tc.payload), tc.field); got != tc.want {
				t.Fatalf("payloadStringField() = %q, want %q", got, tc.want)
			}
		})
	}
}
