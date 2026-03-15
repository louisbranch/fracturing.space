package session

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestFoldSessionStartedSetsFields(t *testing.T) {
	state := State{Ended: true}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("session.started"),
		PayloadJSON: []byte(`{"session_id":"sess-1","session_name":"Chapter One"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.Started {
		t.Fatal("expected session to be started")
	}
	if updated.Ended {
		t.Fatal("expected session to be marked not ended")
	}
	if updated.SessionID != "sess-1" {
		t.Fatalf("session id = %s, want %s", updated.SessionID, "sess-1")
	}
	if updated.Name != "Chapter One" {
		t.Fatalf("session name = %s, want %s", updated.Name, "Chapter One")
	}
}

func TestFoldSessionEndedMarksEnded(t *testing.T) {
	state := State{Started: true, SessionID: "sess-1"}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("session.ended"),
		PayloadJSON: []byte(`{"session_id":"sess-1"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Started {
		t.Fatal("expected session to be marked not started")
	}
	if !updated.Ended {
		t.Fatal("expected session to be marked ended")
	}
}

func TestFoldSessionGateOpenedSetsGateState(t *testing.T) {
	state := State{}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("session.gate_opened"),
		PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"decision","metadata":{"eligible_participant_ids":["p2","p1"],"topic":"direction"}}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.GateOpen {
		t.Fatal("expected gate to be open")
	}
	if updated.GateID != "gate-1" {
		t.Fatalf("gate id = %s, want %s", updated.GateID, "gate-1")
	}
	if updated.GateType != "decision" {
		t.Fatalf("gate type = %s, want decision", updated.GateType)
	}
	if string(updated.GateMetadataJSON) != `{"eligible_participant_ids":["p1","p2"],"topic":"direction"}` {
		t.Fatalf("gate metadata = %s", string(updated.GateMetadataJSON))
	}
}

func TestFoldSessionGateOpenedAcceptsGenericWorkflowMetadata(t *testing.T) {
	state := State{}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("session.gate_opened"),
		PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"decision","metadata":{"topic":"check-in"}}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(updated.GateMetadataJSON) != `{"topic":"check-in"}` {
		t.Fatalf("gate metadata = %s", string(updated.GateMetadataJSON))
	}
}

func TestFoldSessionGateResponseRecordedLeavesGateLifecycleStateUnchanged(t *testing.T) {
	state := State{
		GateOpen:         true,
		GateID:           "gate-1",
		GateType:         "decision",
		GateMetadataJSON: []byte(`{"eligible_participant_ids":["part-1","part-2"]}`),
	}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("session.gate_response_recorded"),
		PayloadJSON: []byte(`{"gate_id":"gate-1","participant_id":"part-1","decision":"ready"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.GateOpen || updated.GateID != state.GateID || updated.GateType != state.GateType {
		t.Fatalf("updated gate lifecycle = %#v, want %#v", updated, state)
	}
	if string(updated.GateMetadataJSON) != string(state.GateMetadataJSON) {
		t.Fatalf("gate metadata = %s, want %s", updated.GateMetadataJSON, state.GateMetadataJSON)
	}
}

func TestFoldSessionGateResolvedClearsGateState(t *testing.T) {
	state := State{GateOpen: true, GateID: "gate-1", GateType: "decision", GateMetadataJSON: []byte(`{"topic":"direction"}`)}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("session.gate_resolved"),
		PayloadJSON: []byte(`{"gate_id":"gate-1","decision":"approve"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.GateOpen {
		t.Fatal("expected gate to be closed")
	}
	if updated.GateID != "" {
		t.Fatalf("gate id = %s, want empty", updated.GateID)
	}
	if updated.GateType != "" || len(updated.GateMetadataJSON) != 0 {
		t.Fatalf("expected gate workflow state to be cleared, got %#v", updated)
	}
}

func TestFoldSessionGateAbandonedClearsGateState(t *testing.T) {
	state := State{GateOpen: true, GateID: "gate-1", GateType: "decision", GateMetadataJSON: []byte(`{"topic":"direction"}`)}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("session.gate_abandoned"),
		PayloadJSON: []byte(`{"gate_id":"gate-1","reason":"timeout"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.GateOpen {
		t.Fatal("expected gate to be closed")
	}
	if updated.GateID != "" {
		t.Fatalf("gate id = %s, want empty", updated.GateID)
	}
	if updated.GateType != "" || len(updated.GateMetadataJSON) != 0 {
		t.Fatalf("expected gate workflow state to be cleared, got %#v", updated)
	}
}

func TestFoldSessionSpotlightSetUpdatesState(t *testing.T) {
	state := State{}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("session.spotlight_set"),
		PayloadJSON: []byte(`{"spotlight_type":"character","character_id":"char-1"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.SpotlightType != "character" {
		t.Fatalf("spotlight type = %s, want %s", updated.SpotlightType, "character")
	}
	if updated.SpotlightCharacterID != "char-1" {
		t.Fatalf("spotlight character id = %s, want %s", updated.SpotlightCharacterID, "char-1")
	}
}

func TestFoldSessionSpotlightClearedResetsState(t *testing.T) {
	state := State{SpotlightType: "gm", SpotlightCharacterID: "char-1"}
	updated, err := Fold(state, event.Event{
		Type:        event.Type("session.spotlight_cleared"),
		PayloadJSON: []byte(`{"reason":"scene change"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.SpotlightType != "" {
		t.Fatalf("spotlight type = %s, want empty", updated.SpotlightType)
	}
	if updated.SpotlightCharacterID != "" {
		t.Fatalf("spotlight character id = %s, want empty", updated.SpotlightCharacterID)
	}
}

func TestFoldRecognizedEventsRejectMalformedPayloads(t *testing.T) {
	tests := []event.Type{
		event.Type("session.started"),
		event.Type("session.ended"),
		event.Type("session.gate_opened"),
		event.Type("session.spotlight_set"),
	}

	for _, eventType := range tests {
		eventType := eventType
		t.Run(string(eventType), func(t *testing.T) {
			_, err := Fold(State{}, event.Event{
				Type:        eventType,
				PayloadJSON: []byte(`{`),
			})
			if err == nil {
				t.Fatal("expected payload decode error")
			}
		})
	}
}
