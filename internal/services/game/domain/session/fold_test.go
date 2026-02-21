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
		PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"gm_consequence"}`),
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
}

func TestFoldSessionGateResolvedClearsGateState(t *testing.T) {
	state := State{GateOpen: true, GateID: "gate-1"}
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
}

func TestFoldSessionGateAbandonedClearsGateState(t *testing.T) {
	state := State{GateOpen: true, GateID: "gate-1"}
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
