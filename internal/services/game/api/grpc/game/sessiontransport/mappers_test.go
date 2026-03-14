package sessiontransport

import (
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestSessionStatusToProto(t *testing.T) {
	tests := []struct {
		name   string
		input  session.Status
		expect campaignv1.SessionStatus
	}{
		{"active", session.StatusActive, campaignv1.SessionStatus_SESSION_ACTIVE},
		{"ended", session.StatusEnded, campaignv1.SessionStatus_SESSION_ENDED},
		{"unspecified", session.StatusUnspecified, campaignv1.SessionStatus_SESSION_STATUS_UNSPECIFIED},
		{"unknown", session.Status("invalid"), campaignv1.SessionStatus_SESSION_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SessionStatusToProto(tt.input); got != tt.expect {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestSessionToProtoEndedAt(t *testing.T) {
	started := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	updated := started.Add(time.Hour)
	ended := started.Add(2 * time.Hour)

	withEnd := SessionToProto(storage.SessionRecord{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Name:       "Session",
		Status:     session.StatusEnded,
		StartedAt:  started,
		UpdatedAt:  updated,
		EndedAt:    &ended,
	})
	if withEnd.GetEndedAt().AsTime().UTC() != ended {
		t.Fatal("expected ended_at to be set")
	}

	noEnd := SessionToProto(storage.SessionRecord{
		ID:         "sess-2",
		CampaignID: "camp-1",
		Name:       "Active",
		Status:     session.StatusActive,
		StartedAt:  started,
		UpdatedAt:  updated,
	})
	if noEnd.GetEndedAt() != nil {
		t.Fatal("expected ended_at to be nil")
	}
}

func TestActiveUserSessionToProto(t *testing.T) {
	started := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	proto := ActiveUserSessionToProto(
		storage.CampaignRecord{ID: "camp-1", Name: "Campaign"},
		storage.SessionRecord{ID: "sess-1", Name: "Session", StartedAt: started},
	)
	if proto.GetCampaignId() != "camp-1" || proto.GetSessionId() != "sess-1" {
		t.Fatalf("unexpected active user session proto: %#v", proto)
	}
	if proto.GetStartedAt().AsTime().UTC() != started {
		t.Fatal("expected started_at to match")
	}
}

func TestTimestampOrNil(t *testing.T) {
	if timestampOrNil(nil) != nil {
		t.Fatal("expected nil timestamp for nil time")
	}
	value := time.Date(2026, 2, 1, 10, 0, 0, 0, time.FixedZone("offset", 3600))
	stamp := timestampOrNil(&value)
	if stamp.AsTime().UTC() != value.UTC() {
		t.Fatal("expected timestamp to be UTC")
	}
}

func TestGateStatusToProto(t *testing.T) {
	tests := []struct {
		name   string
		input  session.GateStatus
		expect campaignv1.SessionGateStatus
	}{
		{"open", session.GateStatusOpen, campaignv1.SessionGateStatus_SESSION_GATE_OPEN},
		{"resolved", session.GateStatusResolved, campaignv1.SessionGateStatus_SESSION_GATE_RESOLVED},
		{"abandoned", session.GateStatusAbandoned, campaignv1.SessionGateStatus_SESSION_GATE_ABANDONED},
		{"open_uppercase", session.GateStatus(" OPEN "), campaignv1.SessionGateStatus_SESSION_GATE_OPEN},
		{"empty", session.GateStatus(""), campaignv1.SessionGateStatus_SESSION_GATE_STATUS_UNSPECIFIED},
		{"unknown", session.GateStatus("invalid"), campaignv1.SessionGateStatus_SESSION_GATE_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GateStatusToProto(tt.input); got != tt.expect {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestSpotlightTypeFromProto(t *testing.T) {
	tests := []struct {
		name      string
		input     campaignv1.SessionSpotlightType
		expect    session.SpotlightType
		wantError bool
	}{
		{"gm", campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM, session.SpotlightTypeGM, false},
		{"character", campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER, session.SpotlightTypeCharacter, false},
		{"unspecified", campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_UNSPECIFIED, "", true},
		{"unknown", campaignv1.SessionSpotlightType(99), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SpotlightTypeFromProto(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expect {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestSpotlightTypeToProto(t *testing.T) {
	tests := []struct {
		name   string
		input  session.SpotlightType
		expect campaignv1.SessionSpotlightType
	}{
		{"gm", session.SpotlightTypeGM, campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM},
		{"character", session.SpotlightTypeCharacter, campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER},
		{"gm_uppercase", session.SpotlightType(" GM "), campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM},
		{"empty", session.SpotlightType(""), campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_UNSPECIFIED},
		{"unknown", session.SpotlightType("invalid"), campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SpotlightTypeToProto(tt.input); got != tt.expect {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestGateToProto(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	resolved := created.Add(time.Hour)

	gate, err := GateToProto(storage.SessionGate{
		GateID:              "gate-1",
		CampaignID:          "camp-1",
		SessionID:           "sess-1",
		GateType:            "decision",
		Status:              "open",
		Reason:              "test",
		CreatedAt:           created,
		CreatedByActorType:  "user",
		CreatedByActorID:    "user-1",
		ResolvedAt:          &resolved,
		ResolvedByActorType: "user",
		ResolvedByActorID:   "user-2",
		Metadata:            map[string]any{"key": "val"},
		Resolution:          map[string]any{"choice": "yes"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate.GetId() != "gate-1" {
		t.Fatalf("expected gate id gate-1, got %v", gate.GetId())
	}
	if gate.GetStatus() != campaignv1.SessionGateStatus_SESSION_GATE_OPEN {
		t.Fatalf("expected open status, got %v", gate.GetStatus())
	}
	if gate.GetMetadata().AsMap()["key"] != "val" {
		t.Fatal("expected metadata key=val")
	}
	if gate.GetResolution().AsMap()["choice"] != "yes" {
		t.Fatal("expected resolution choice=yes")
	}

	gate2, err := GateToProto(storage.SessionGate{
		GateID:     "gate-2",
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		GateType:   "decision",
		Status:     "resolved",
		CreatedAt:  created,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gate2.GetMetadata() != nil {
		t.Fatal("expected nil metadata")
	}

	_, err = GateToProto(storage.SessionGate{
		GateID:    "gate-3",
		Metadata:  map[string]any{"bad": make(chan int)},
		CreatedAt: created,
	})
	if err == nil {
		t.Fatal("expected error for invalid metadata payload")
	}

	_, err = GateToProto(storage.SessionGate{
		GateID:     "gate-4",
		Resolution: map[string]any{"bad": make(chan int)},
		CreatedAt:  created,
	})
	if err == nil {
		t.Fatal("expected error for invalid resolution payload")
	}
}

func TestGateToProtoNormalizesWorkflowMetadataArraysForStructPB(t *testing.T) {
	created := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	metadata, err := session.DecodeGateMetadataMap(session.GateTypeReadyCheck, []byte(`{"eligible_participant_ids":["p2","p1"],"options":["wait","ready"]}`))
	if err != nil {
		t.Fatalf("DecodeGateMetadataMap() error = %v", err)
	}

	gate, err := GateToProto(storage.SessionGate{
		GateID:     "gate-typed",
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		GateType:   session.GateTypeReadyCheck,
		Status:     session.GateStatusOpen,
		CreatedAt:  created,
		Metadata:   metadata,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	options, ok := gate.GetMetadata().AsMap()["options"].([]any)
	if !ok {
		t.Fatalf("metadata options type = %T, want []any", gate.GetMetadata().AsMap()["options"])
	}
	if len(options) != 2 || options[0] != "ready" || options[1] != "wait" {
		t.Fatalf("metadata options = %#v", options)
	}
}

func TestSpotlightToProto(t *testing.T) {
	updated := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)

	spotlight := SpotlightToProto(storage.SessionSpotlight{
		CampaignID:         "camp-1",
		SessionID:          "sess-1",
		SpotlightType:      "gm",
		CharacterID:        "",
		UpdatedAt:          updated,
		UpdatedByActorType: "user",
		UpdatedByActorID:   "user-1",
	})
	if spotlight.GetType() != campaignv1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM {
		t.Fatalf("expected gm spotlight type, got %v", spotlight.GetType())
	}
	if spotlight.GetCampaignId() != "camp-1" {
		t.Fatalf("expected camp-1, got %v", spotlight.GetCampaignId())
	}
}
