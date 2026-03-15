package outcometransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestHandlerApplyRollOutcomeIdempotentFearRepairsGMConsequence(t *testing.T) {
	handler, events, recorder := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{
		requestID: "roll-fear-1",
		outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		metadata: workflowtransport.RollSystemMetadata{
			CharacterID: "char-1",
			RollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			HopeFear:    workflowtransport.BoolPtr(true),
			GMMove:      workflowtransport.BoolPtr(true),
		},
	})
	if _, err := events.AppendEvent(context.Background(), event.Event{
		CampaignID: "camp-1",
		Timestamp:  testTimestamp,
		Type:       eventTypeActionOutcomeApplied,
		SessionID:  "sess-1",
		RequestID:  "roll-fear-1",
		EntityType: "outcome",
		EntityID:   "roll-fear-1",
	}); err != nil {
		t.Fatalf("append outcome event: %v", err)
	}

	resp, err := handler.ApplyRollOutcome(testSessionContext("camp-1", "sess-1"), &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		SceneId:   "scene-1",
		RollSeq:   roll.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if !resp.GetRequiresComplication() {
		t.Fatal("expected complication on fear retry")
	}
	if got := len(recorder.coreCommands); got != 2 {
		t.Fatalf("core command count = %d, want 2", got)
	}
	if recorder.coreCommands[0].CommandType != commandTypeSessionGateOpen {
		t.Fatalf("first core command type = %q", recorder.coreCommands[0].CommandType)
	}
	if recorder.coreCommands[1].CommandType != commandTypeSessionSpotlightSet {
		t.Fatalf("second core command type = %q", recorder.coreCommands[1].CommandType)
	}
}
