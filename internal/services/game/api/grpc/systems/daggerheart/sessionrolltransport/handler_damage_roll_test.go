package sessionrolltransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestHandlerSessionDamageRollSuccess(t *testing.T) {
	var resolveCalls int
	handler := newTestHandler(Dependencies{
		ExecuteDamageRollResolve: func(_ context.Context, in RollResolveInput) (uint64, error) {
			resolveCalls++
			if in.MissingEventMsg != "damage roll did not emit an event" {
				t.Fatalf("missing event msg = %q", in.MissingEventMsg)
			}
			return 8, nil
		},
	})

	resp, err := handler.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Dice:        []*pb.DiceSpec{{Sides: 6, Count: 2}},
		Modifier:    1,
	})
	if err != nil {
		t.Fatalf("SessionDamageRoll returned error: %v", err)
	}
	if resp.GetRollSeq() != 8 {
		t.Fatalf("roll_seq = %d, want 8", resp.GetRollSeq())
	}
	if resolveCalls != 1 {
		t.Fatalf("resolve calls = %d, want 1", resolveCalls)
	}
}
