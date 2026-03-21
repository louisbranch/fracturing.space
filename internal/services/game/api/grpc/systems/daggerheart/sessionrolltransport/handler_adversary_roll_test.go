package sessionrolltransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestHandlerSessionAdversaryAttackRollSuccess(t *testing.T) {
	var loaded bool
	handler := newTestHandler(Dependencies{
		ExecuteAdversaryRollResolve: func(_ context.Context, in RollResolveInput) (uint64, error) {
			if in.EntityType != "adversary" || in.EntityID != "adv-1" {
				t.Fatalf("resolve entity = %s/%s", in.EntityType, in.EntityID)
			}
			return 6, nil
		},
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			loaded = true
			return projectionstore.DaggerheartAdversary{AdversaryID: "adv-1", SessionID: "sess-1"}, nil
		},
	})

	resp, err := handler.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		Modifiers:   []*pb.ActionRollModifier{{Source: "attack", Value: 2}},
		Advantage:   1,
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}
	if !loaded {
		t.Fatal("expected adversary loader to be called")
	}
	if resp.GetRollSeq() != 6 {
		t.Fatalf("roll_seq = %d, want 6", resp.GetRollSeq())
	}
}

func TestHandlerSessionAdversaryActionCheckAutoSuccess(t *testing.T) {
	handler := newTestHandler(Dependencies{
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{AdversaryID: "adv-1", SessionID: "sess-1"}, nil
		},
	})

	resp, err := handler.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		Difficulty:  10,
		Modifiers:   []*pb.ActionRollModifier{{Source: "action", Value: 2}},
		Dramatic:    false,
	})
	if err != nil {
		t.Fatalf("SessionAdversaryActionCheck returned error: %v", err)
	}
	if !resp.GetAutoSuccess() {
		t.Fatal("expected auto success")
	}
	if resp.GetRollSeq() != 5 {
		t.Fatalf("roll_seq = %d, want 5", resp.GetRollSeq())
	}
	if resp.GetTotal() != 2 {
		t.Fatalf("total = %d, want 2", resp.GetTotal())
	}
}
