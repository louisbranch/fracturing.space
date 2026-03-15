package outcometransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
)

func TestHandlerApplyAdversaryAttackOutcomeSuccess(t *testing.T) {
	handler, events, _ := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{
		outcome: pb.Outcome_SUCCESS_WITH_FEAR.String(),
		metadata: workflowtransport.RollSystemMetadata{
			CharacterID: "adv-1",
			AdversaryID: "adv-1",
			RollKind:    "adversary_roll",
			Roll:        workflowtransport.IntPtr(20),
			Modifier:    workflowtransport.IntPtr(2),
			Total:       workflowtransport.IntPtr(22),
		},
	})

	resp, err := handler.ApplyAdversaryAttackOutcome(testSessionContext("camp-1", "sess-1"), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  "sess-1",
		RollSeq:    roll.Seq,
		Difficulty: 18,
		Targets:    []string{"char-1"},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryAttackOutcome returned error: %v", err)
	}
	if !resp.GetResult().GetSuccess() {
		t.Fatal("expected successful adversary attack")
	}
	if !resp.GetResult().GetCrit() {
		t.Fatal("expected crit from natural 20")
	}
}
