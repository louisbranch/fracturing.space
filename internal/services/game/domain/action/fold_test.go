package action

import (
	"encoding/json"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestFold_RollResolvedMutatesCausalReplayState(t *testing.T) {
	state := State{}
	payloadJSON, err := json.Marshal(RollResolvePayload{
		RequestID: "req-1",
		RollSeq:   42,
		Outcome:   "SUCCESS_WITH_HOPE",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	updated, err := Fold(state, event.Event{
		Type:        EventTypeRollResolved,
		SessionID:   "sess-1",
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := updated.Rolls[42]; !ok {
		t.Fatal("expected roll state for roll seq 42")
	}
	if updated.Rolls[42].RequestID != "req-1" {
		t.Fatalf("request id = %s, want req-1", updated.Rolls[42].RequestID)
	}
}

func TestFold_OutcomeAppliedMutatesCausalReplayState(t *testing.T) {
	state := State{}
	payloadJSON, err := json.Marshal(OutcomeApplyPayload{
		RequestID: "req-1",
		RollSeq:   99,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	updated, err := Fold(state, event.Event{
		Type:        EventTypeOutcomeApplied,
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := updated.AppliedOutcomes[99]; !ok {
		t.Fatal("expected applied outcome for roll seq 99")
	}
}

// TestFold_AuditOnlyEventsAreNotHandled verifies that audit-only event types
// (outcome_rejected, note_added) are not registered in the fold router.
// These events are filtered out by the aggregate folder before reaching
// domain folds — a fold handler for them would be dead code.
func TestFold_AuditOnlyEventsAreNotHandled(t *testing.T) {
	auditOnlyTypes := []event.Type{EventTypeOutcomeRejected, EventTypeNoteAdded}
	for _, evtType := range auditOnlyTypes {
		_, err := Fold(State{}, event.Event{Type: evtType, PayloadJSON: []byte(`{}`)})
		if err == nil {
			t.Fatalf("expected error for audit-only event type %s, got nil", evtType)
		}
	}
}
