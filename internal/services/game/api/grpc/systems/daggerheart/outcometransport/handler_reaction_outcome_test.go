package outcometransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
)

func TestHandlerApplyReactionOutcomeCritNegatesEffects(t *testing.T) {
	handler, events, _ := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{
		outcome: pb.Outcome_CRITICAL_SUCCESS.String(),
		metadata: workflowtransport.RollSystemMetadata{
			CharacterID: "char-1",
			RollKind:    pb.RollKind_ROLL_KIND_REACTION.String(),
			Crit:        workflowtransport.BoolPtr(true),
		},
	})

	resp, err := handler.ApplyReactionOutcome(testSessionContext("camp-1", "sess-1"), &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   roll.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyReactionOutcome returned error: %v", err)
	}
	if !resp.GetResult().GetCrit() {
		t.Fatal("expected crit result")
	}
	if !resp.GetResult().GetEffectsNegated() {
		t.Fatal("expected crit to negate effects")
	}
}
