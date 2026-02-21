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

	updated := Fold(state, event.Event{
		Type:        eventTypeRollResolved,
		SessionID:   "sess-1",
		PayloadJSON: payloadJSON,
	})
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

	updated := Fold(state, event.Event{
		Type:        eventTypeOutcomeApplied,
		PayloadJSON: payloadJSON,
	})
	if _, ok := updated.AppliedOutcomes[99]; !ok {
		t.Fatal("expected applied outcome for roll seq 99")
	}
}

func TestFold_NonCausalEventsDoNotMutateCausalReplayState(t *testing.T) {
	base := State{
		Rolls: map[uint64]RollState{
			7: {RequestID: "req-7", SessionID: "sess-1", Outcome: "SUCCESS_WITH_HOPE"},
		},
		AppliedOutcomes: map[uint64]struct{}{
			7: {},
		},
	}
	unchanged := Fold(base, event.Event{Type: eventTypeOutcomeRejected, PayloadJSON: []byte(`{"roll_seq":7}`)})
	if len(unchanged.Rolls) != 1 {
		t.Fatalf("rolls size = %d, want 1", len(unchanged.Rolls))
	}
	if len(unchanged.AppliedOutcomes) != 1 {
		t.Fatalf("applied outcomes size = %d, want 1", len(unchanged.AppliedOutcomes))
	}

	unchanged = Fold(base, event.Event{Type: eventTypeNoteAdded, PayloadJSON: []byte(`{"content":"note"}`)})
	if len(unchanged.Rolls) != 1 {
		t.Fatalf("rolls size = %d, want 1", len(unchanged.Rolls))
	}
	if len(unchanged.AppliedOutcomes) != 1 {
		t.Fatalf("applied outcomes size = %d, want 1", len(unchanged.AppliedOutcomes))
	}
}
