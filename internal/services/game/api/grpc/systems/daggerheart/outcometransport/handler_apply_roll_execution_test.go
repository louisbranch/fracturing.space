package outcometransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
)

func TestHandlerApplyRollOutcomeSuccess(t *testing.T) {
	handler, events, recorder := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{
		requestID: "roll-hope-1",
		outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		metadata: workflowtransport.RollSystemMetadata{
			CharacterID: "char-1",
			RollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			HopeFear:    workflowtransport.BoolPtr(true),
			GMMove:      workflowtransport.BoolPtr(false),
		},
	})

	resp, err := handler.ApplyRollOutcome(testSessionContext("camp-1", "sess-1"), &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   roll.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if got := resp.GetUpdated().GetCharacterStates()[0].GetHope(); got != 3 {
		t.Fatalf("updated hope = %d, want 3", got)
	}
	if got := len(recorder.systemCommands); got != 1 {
		t.Fatalf("system command count = %d, want 1", got)
	}
	if got := recorder.systemCommands[0].CommandType; got != commandTypeDaggerheartCharacterStatePatch {
		t.Fatalf("system command type = %q", got)
	}
	if got := len(recorder.coreCommands); got != 1 {
		t.Fatalf("core command count = %d, want 1", got)
	}
	if got := recorder.coreCommands[0].CommandType; got != commandTypeActionOutcomeApply {
		t.Fatalf("core command type = %q", got)
	}
	if got := len(recorder.stressCalls); got != 1 {
		t.Fatalf("stress call count = %d, want 1", got)
	}
}
