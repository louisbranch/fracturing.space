package outcometransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
)

func TestHandlerApplyAttackOutcomeSuccess(t *testing.T) {
	handler, events, _ := newTestHandler()
	roll := appendRollEvent(t, events, rollEventConfig{
		outcome: pb.Outcome_SUCCESS_WITH_HOPE.String(),
		metadata: workflowtransport.RollSystemMetadata{
			CharacterID: "char-1",
			RollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			HopeFear:    workflowtransport.BoolPtr(true),
		},
	})

	resp, err := handler.ApplyAttackOutcome(testSessionContext("camp-1", "sess-1"), &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   roll.Seq,
		Targets:   []string{"char-2"},
	})
	if err != nil {
		t.Fatalf("ApplyAttackOutcome returned error: %v", err)
	}
	if got := resp.GetCharacterId(); got != "char-1" {
		t.Fatalf("character_id = %q, want char-1", got)
	}
	if got := resp.GetResult().GetFlavor(); got != "HOPE" {
		t.Fatalf("flavor = %q, want HOPE", got)
	}
	if !resp.GetResult().GetSuccess() {
		t.Fatal("expected success result")
	}
}
